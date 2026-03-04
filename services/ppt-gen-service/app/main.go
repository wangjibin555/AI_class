package main

import (
	"AI_class/def"
	"AI_class/infrastructure/minio"
	"AI_class/infrastructure/mysql"
	"AI_class/infrastructure/redis"
	"AI_class/pkg/config"
	"AI_class/pkg/config/namespace/database"
	"AI_class/pkg/config/namespace/mq"
	"AI_class/pkg/config/namespace/server"
	"AI_class/pkg/graceful"
	"AI_class/pkg/logger"
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	_ "runtime/pprof"
	"syscall"
	"time"

	"github.com/cloudflare/tableflip"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// 依赖容器
type Dependencies struct {
	MysqlClient    *mysql.MysqlClient
	RedisClient    *redis.RedisClient
	MinioClient    *minio.MinioClient
	RabbitMQClient *mq.RabbitMQClient
}

// 普罗米修斯指标定义
var (
	//HTTP请求总数
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of http request",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestsDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_requests_duration_seconds",
			Help:    "Duration of http requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	httpRequestsInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Number of http requests in flight",
		},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestsDuration)
	prometheus.MustRegister(httpRequestsInFlight)
}

func main() {
	go func() {
		initPprof()
	}()

	upg, err := tableflip.New(tableflip.Options{})
	if err != nil {
		panic(err)
	}
	defer upg.Stop()

	ctx := context.Background()

	//初始化配置中心
	if err := initConfig(ctx); err != nil {
		logger.Error("Failed to init config: %v", err)
	}

	//初始化基础设施
	deps, err := initInfrastructure(ctx)
	if err != nil {
		logger.Error("Failed to init config: %v", err)
	}

	//构建Gin服务器
	ginEngine := buildGinServer(ctx, deps)

	//创建HTTP服务器
	httpServer := &http.Server{
		Handler:           ginEngine,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		ReadHeaderTimeout: 45 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	ln, err := upg.Listen("tcp", getListenAddress(ctx))
	if err != nil {
		logger.Error("Failed to listen: %v", err)
	}

	go func() {
		logger.Info("HTTP server starting on %s", ln.Addr().String())
		if err := httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error: %v", err)
		}
	}()

	//注册HTTP服务器关闭函数
	graceful.RegisterAllFunc(func() {
		logger.Info("Shutting down HTTP server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err = httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("Failed to shutdown HTTP server: %v", err)
		}
	})

	//通知tableflip准备就绪
	if err = upg.Ready(); err != nil {
		panic(err)
	}
	logger.Info("Service ready, PID: %d", os.Getpid())

	//等待退出信号
	waitForShutdown(upg)

	//执行平滑退出信号
	logger.Info("Service down service...")
	graceful.ClearAllFunc(30 * time.Second)

	logger.Info("Service ClearAllFunc success...")

}

// Gin服务构建
func buildGinServer(ctx context.Context, deps *Dependencies) *gin.Engine {
	//设置Gin
	cfg, _ := config.Get("server")
	if cfg != nil {
		serverConfig := cfg.(*server.ServerConfig)
		if serverConfig.HTTP.Port != 8081 {
			gin.SetMode(gin.ReleaseMode)
		}
	}

	//创建Gin引擎
	r := gin.New()
	//全局中间件
	r.Use(gin.Recovery())
	r.Use(LoggerMiddleware())
	r.Use(CorsMiddleware())
	r.Use(AdminAuthMiddleware())
	r.Use(PrometheusMiddleware())

	//Promethus监测端点
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	//健康检查路由组
	health := r.Group("/health")
	{
		health.GET("", HealthCheck)
		health.GET("/ready", ReadyCheck)
		health.GET("/live", LiveCheck)
	}
	return r
}

// 初始化配置中心（包含配置中心所需的Redis,Mysql）
func initConfig(ctx context.Context) error {
	return config.Bootstrap(ctx,
		config.WithConfigpath("./config"),
		config.WithEnvionment("dev"),
		config.WithEnableCache(true),
		config.WithCacheOptions(
			config.WithMaxSize(2000),
			config.WithEvictionPolicy(config.EvictionPolicyLRU),
		),
		config.WithAutoRegisterShutdown(true),
		config.WithShutdownTimeout(def.ShutdownTimeout),
	)
}

type MysqlConfig struct {
	//基础配置
	Host     string
	Port     int
	UserName string
	Password string
	Database string
	Charset  string

	//连接池配置
	MaxIdleconns    int           //最大空闲空闲连接数
	MaxOpenConns    int           //最大连接数
	ConnMaxLifetime time.Duration //最大连接生命周期
	ConnMaxIdletime time.Duration //最大空闲连接生命周期

	//超时配置
	ConnTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	//其他配置
	LogLevel      string        //日志级别
	SlowThreshold time.Duration //慢查询阈值
}

func initInfrastructure(ctx context.Context) (*Dependencies, error) {
	deps := &Dependencies{}

	//配置中心配置
	cfg, err := config.Get("database")
	if err != nil {
		return nil, fmt.Errorf("failed to get database config: %w", err)
	}
	dbConfig := cfg.(*database.DataBaseConfig)

	//初始化Mysql
	mysqlClient, err := mysql.NewMysqlClient(&mysql.MysqlConfig{
		Host:            dbConfig.Mysql.GetHost(),
		Port:            dbConfig.Mysql.GetPort(),
		UserName:        dbConfig.Mysql.GetUserName(),
		Password:        dbConfig.Mysql.GetPassword(),
		Database:        dbConfig.Mysql.GetDataBase(),
		Charset:         dbConfig.Mysql.GetCharset(),
		MaxIdleconns:    dbConfig.Mysql.GetMaxIdleConns(),
		MaxOpenConns:    dbConfig.Mysql.GetMaxOpenConns(),
		ConnMaxLifetime: dbConfig.Mysql.GetConnMaxLifetime(),
		ConnMaxIdletime: dbConfig.Mysql.GetConnMaxIdletime(),
		ConnTimeout:     dbConfig.Mysql.GetConnTimeout(),
		ReadTimeout:     dbConfig.Mysql.GetReadTimeout(),
		WriteTimeout:    dbConfig.Mysql.GetWriteTimeout(),
		LogLevel:        dbConfig.Mysql.GetLogLevel(),
		SlowThreshold:   time.Duration(dbConfig.Mysql.GetSlowThreshold()),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init mysql: %w", err)
	}
	deps.MysqlClient = mysqlClient
	logger.Info("Mysql client initialized")

	//Mysql客户端初始化
	graceful.RegisterAllFunc(func() {
		if err = mysqlClient.Close(); err != nil {
			logger.Error("Failed to close mysql client: %v", err)
		}
	})

	redisClient, err := dbConfig.Redis.NewRedisClient()
	if err != nil {
		return nil, fmt.Errorf("failed to init redis: %w", err)
	}
	deps.RedisClient = redisClient
	logger.Info("Redis client initialized")

	graceful.RegisterAllFunc(func() {
		if err = redisClient.ClosePool(); err != nil {
			logger.Error("Failed to close redis: %v", err)
		}
	})

	//初始化Minio
	minioConfig := &minio.MinioConfig{
		Endpoint:        "localhost:9000",
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		UseSSL:          false,
		BucketName:      "ppt-files",
		Location:        "us-east-1",
	}
	minioClient, err := minio.NewMinioClient(minioConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to init minio: %w", err)
	}
	deps.MinioClient = minioClient
	logger.Info("Minio client initialized")

	//初始化RabbitMQ
	rabbitMQConfig := &mq.RabbitMQConfig{
		Host:        "localhost",
		Port:        5672,
		User:        "guest",
		Password:    "guest",
		VirtualHost: "/",
		MaxChannels: 10,
	}
	rabbitMQClient, err := mq.NewRabbitMQClient(rabbitMQConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to init rabbitmq: %w", err)
	}
	deps.RabbitMQClient = rabbitMQClient
	logger.Info("RabbitMQ client initialized")

	graceful.RegisterAllFunc(func() {
		if err = rabbitMQClient.Close(); err != nil {
			logger.Error("Failed to close rabbitmq: %v", err)
		}
	})
	return deps, nil
}

func getListenAddress(ctx context.Context) string {
	cfg, err := config.Get("server")
	if err == nil {
		serverConfig := cfg.(*server.ServerConfig)
		httpConfig := serverConfig.HTTP
		if httpConfig.Host != "" && httpConfig.Port > 0 {
			return fmt.Sprintf("%s:%d", httpConfig.Host, httpConfig.Port)
		}
	}
	return ":8080"
}

// 关闭服务后等待30s，让其他服务关闭
func waitForShutdown(upg *tableflip.Upgrader) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	exitChan := make(chan struct{})

	go func() {
		<-upg.Exit()
		logger.Info("Received tableflip exit signal")
		close(exitChan)
	}()

	go func() {
		sig := <-sigChan
		logger.Info("Received system signal: %v", sig)
		close(exitChan)
	}()

	<-exitChan
	logger.Info("Exit signal received, starting graceful shutdown...")
}

// Gin日志中间件
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		if raw != "" {
			path = path + "?" + raw
		}

		logger.Info("[GIN] %s | %3d | %13v | %15s | %-7s %s %s",
			time.Now().Format("2006/01/02 - 15:04:05"),
			statusCode,
			latency,
			clientIP,
			method,
			path,
			errorMessage,
		)
	}
}

// Cors中间件 (相对固定)
func CorsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

// 登陆Token认证
func AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取请求头中的token
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization token"})
			c.Abort()
			return
		}
	}
}

// 普罗米修斯实现
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}
		start := time.Now()

		//增加正在请求的数量
		httpRequestsInFlight.Inc()
		defer httpRequestsInFlight.Dec()

		//进行处理请求
		c.Next()

		//记录请求持续时间
		duration := time.Since(start).Seconds()
		httpRequestsDuration.WithLabelValues(c.Request.Method, c.FullPath()).Observe(duration)

		//记录请求总数
		httpRequestsTotal.WithLabelValues(c.Request.Method, c.FullPath(), fmt.Sprintf("%d", c.Writer.Status())).Inc()
	}
}

// ============ 健康检查处理器 ============
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func ReadyCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

func LiveCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "alive"})
}

func initPprof() {
	defer func() {
		//panic试图恢复
		if r := recover(); r != nil {
			time.AfterFunc(time.Second*5, func() {
				initPprof()
			})
		}
	}()
	address := ":7077"
	err := http.ListenAndServe(address, nil)
	if err != nil {
		logger.Info("start profile api err", err.Error())
		time.AfterFunc(time.Second*5, func() {
			initPprof()
		})
	} else {
		logger.Info("start profile api success")
	}
}

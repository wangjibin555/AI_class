package coze

import "time"

// HTML转换相关的类型定义

// PPTSlide HTML幻灯片结构（用于reveal.js）
type HTMLPPTSlide struct {
	ID         string           `json:"id"`
	Title      string           `json:"title"`
	Content    string           `json:"content"`
	Background string           `json:"background"`
	Transition string           `json:"transition"`
	Duration   int              `json:"duration"`
	Notes      string           `json:"notes"`
	Elements   []SlideElement   `json:"elements"`
	Animations []SlideAnimation `json:"animations"`
	Layout     string           `json:"layout"`
}

// SlideElement 幻灯片元素
type SlideElement struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"` // text, image, chart, video, shape
	Content    string                 `json:"content"`
	Position   ElementPosition        `json:"position"`
	Style      map[string]interface{} `json:"style"`
	Animation  ElementAnimation       `json:"animation"`
	Attributes map[string]string      `json:"attributes"`
}

// ElementPosition 元素位置
type ElementPosition struct {
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	ZIndex   int     `json:"z_index"`
	Rotation float64 `json:"rotation"`
}

// ElementAnimation 元素动画
type ElementAnimation struct {
	Type      string `json:"type"`      // fade, slide, zoom, rotate, bounce
	Duration  int    `json:"duration"`  // 毫秒
	Delay     int    `json:"delay"`     // 毫秒
	Easing    string `json:"easing"`    // ease, linear, ease-in, ease-out
	Direction string `json:"direction"` // left, right, up, down
	Repeat    int    `json:"repeat"`    // 重复次数，-1为无限
}

// SlideAnimation 幻灯片动画
type SlideAnimation struct {
	TriggerEvent   string          `json:"trigger_event"` // click, auto, hover
	TargetElements []string        `json:"target_elements"`
	Sequence       []AnimationStep `json:"sequence"`
}

// AnimationStep 动画步骤
type AnimationStep struct {
	ElementID string           `json:"element_id"`
	Animation ElementAnimation `json:"animation"`
	StartTime int              `json:"start_time"` // 相对开始时间
}

// HTMLResult HTML转换结果
type HTMLResult struct {
	LocalPath      string          `json:"local_path"`
	CDNURL         string          `json:"cdn_url"`
	PreviewURL     string          `json:"preview_url"`
	TotalSlides    int             `json:"total_slides"`
	FileSize       int64           `json:"file_size"`
	GeneratedAt    time.Time       `json:"generated_at"`
	Slides         []HTMLPPTSlide  `json:"slides"`
	Resources      []HTMLResource  `json:"resources"`
	Metadata       HTMLMetadata    `json:"metadata"`
	SourceInfo     *TMPFileInfo    `json:"source_info"`     // .tmp文件信息
	ConversionInfo *ConversionInfo `json:"conversion_info"` // 转换信息
}

// HTMLResource HTML资源
type HTMLResource struct {
	Type      string `json:"type"` // css, js, image, font
	LocalPath string `json:"local_path"`
	CDNURL    string `json:"cdn_url"`
	Size      int64  `json:"size"`
	Hash      string `json:"hash"`
}

// HTMLMetadata HTML元数据
type HTMLMetadata struct {
	Title       string    `json:"title"`
	Author      string    `json:"author"`
	Description string    `json:"description"`
	Theme       string    `json:"theme"`
	Language    string    `json:"language"`
	Keywords    []string  `json:"keywords"`
	CreatedAt   time.Time `json:"created_at"`
	ModifiedAt  time.Time `json:"modified_at"`
}

// ConvertOptions 转换选项
type ConvertOptions struct {
	Theme          string `json:"theme"`           // 主题样式
	Transition     string `json:"transition"`      // 切换效果
	AutoPlay       bool   `json:"auto_play"`       // 自动播放
	AutoPlayDelay  int    `json:"auto_play_delay"` // 自动播放延迟
	ShowControls   bool   `json:"show_controls"`   // 显示控制栏
	ShowProgress   bool   `json:"show_progress"`   // 显示进度条
	EnableTouch    bool   `json:"enable_touch"`    // 触摸支持
	EnableKeyboard bool   `json:"enable_keyboard"` // 键盘支持
	CustomCSS      string `json:"custom_css"`      // 自定义CSS
	CustomJS       string `json:"custom_js"`       // 自定义JS
}

// ConvertConfig 转换配置
type ConvertConfig struct {
	CDNBaseURL   string `yaml:"cdn_base_url"`
	TemplateType string `yaml:"template_type"` // reveal, impress, custom
	OutputFormat string `yaml:"output_format"` // html, pdf, images
	EnableCache  bool   `yaml:"enable_cache"`
	CacheTimeout int    `yaml:"cache_timeout"`
	CustomCSS    string `yaml:"custom_css"`
	CustomJS     string `yaml:"custom_js"`
}

// CDNUploader CDN上传器接口
type CDNUploader struct {
	config *ConvertConfig
}

// NewCDNUploader 创建CDN上传器
func NewCDNUploader(config *ConvertConfig) *CDNUploader {
	return &CDNUploader{
		config: config,
	}
}

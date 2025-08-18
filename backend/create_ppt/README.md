# 网页爬取与AI总结PPT生成器

这是一个用Go语言开发的工具，可以爬取网页内容或读取本地文件（PDF、DOCX、TXT），使用通义千问AI进行内容总结，并生成PPT文件。

## 功能特性

- 🌐 **网页爬取**: 自动爬取指定URL的网页内容
- 📄 **文件处理**: 支持读取PDF、DOCX、TXT格式的本地文件
- 📝 **内容提取**: 智能提取网页或文档中的主要文本内容
- 🤖 **AI总结**: 使用通义千问AI对内容进行结构化总结
- 📊 **PPT生成**: 将总结内容生成真正的PPT格式文件（.pptx）

## 支持的文件格式

- **网页URL**: 直接输入网页地址进行爬取
- **PDF文件**: 支持多页PDF文档的文本提取
- **DOCX文件**: 支持Word文档的段落和表格内容提取
- **TXT文件**: 支持纯文本文件的读取

## 安装要求

- Go 1.21 或更高版本
- Python 3.7+ 和 pip3（用于PPT转换和DOCX处理）
- 通义千问API密钥

## 安装步骤

1. 克隆项目到本地：
```bash
git clone <repository-url>
cd web-scraper-ppt
```

2. 安装Go依赖：
```bash
go mod tidy
```

3. 安装Python依赖（用于PPT转换和DOCX处理）：
```bash
./install_python_deps.sh
pip3 install python-docx
```

3. 配置API密钥（选择以下任一方式）：

   **方式1: 环境变量**
   ```bash
   export QIANWEN_API_KEY="your_api_key_here"
   ```

   **方式2: 配置文件**
   编辑 `config.json` 文件：
   ```json
   {
     "qianwen_api_key": "your_api_key_here",
     "output_dir": "output"
   }
   ```

## 使用方法

### 网页爬取模式

运行程序并指定要爬取的URL：

```bash
go run main.go https://example.com
```

### 文件处理模式

运行程序并指定要处理的文件：

```bash
# 处理PDF文件
go run main.go --file document.pdf

# 处理Word文档
go run main.go --file report.docx

# 处理文本文件
go run main.go --file notes.txt
```

程序将自动执行以下步骤：
1. 读取网页内容或文件内容
2. 提取文本内容
3. 使用AI进行总结
4. 生成PPT文件

生成的PPT文件将保存在 `output` 目录中，文件名格式根据输入类型而定：
- 网页内容：`网页内容总结_YYYYMMDD_HHMMSS.pptx`
- PDF文档：`PDF文档总结_YYYYMMDD_HHMMSS.pptx`
- Word文档：`Word文档总结_YYYYMMDD_HHMMSS.pptx`
- 文本文件：`文本文件总结_YYYYMMDD_HHMMSS.pptx`

## 获取通义千问API密钥

1. 访问 [阿里云通义千问](https://dashscope.console.aliyun.com/)
2. 注册并登录账号
3. 创建API密钥
4. 将密钥配置到环境变量或配置文件中

## 项目结构

```
web-scraper-ppt/
├── main.go                    # 主程序文件
├── ppt_generator.go           # PPT生成器
├── file_processor.go          # 文件处理器
├── convert_to_ppt.py          # Python PPT转换脚本
├── install_python_deps.sh     # Python依赖安装脚本
├── test_file_processing.sh    # 文件处理测试脚本
├── go.mod                     # Go模块文件
├── go.sum                     # 依赖校验文件
├── config.json                # 配置文件
├── README.md                  # 项目说明
├── example.sh                 # 示例使用脚本
├── test.sh                    # 测试脚本
└── output/                    # 输出目录（自动创建）
```

## 技术栈

- **Go**: 主要编程语言
- **goquery**: HTML解析和内容提取
- **excelize**: Excel文件生成
- **python-docx**: DOCX文件处理
- **pdf**: PDF文件处理
- **Python**: PPT格式转换
- **openpyxl**: Excel文件读取
- **python-pptx**: PPT文件生成
- **通义千问API**: AI内容总结

## 注意事项

1. 确保有足够的通义千问API调用额度
2. 网页内容长度有限制（最大8000字符）
3. 生成的PPT文件为真正的PPT格式（.pptx），可以用PowerPoint打开
4. 需要安装Python依赖才能生成PPT格式文件
5. 建议在合法合规的前提下使用，遵守网站的robots.txt规则
6. PDF文件处理需要文件包含可提取的文本内容
7. DOCX文件支持段落和表格内容的提取

## 故障排除

### 常见问题

1. **API密钥错误**
   - 检查API密钥是否正确设置
   - 确认API密钥有足够的调用额度

2. **网页爬取失败**
   - 检查URL是否正确
   - 确认网络连接正常
   - 某些网站可能有反爬虫机制

3. **文件读取失败**
   - 检查文件路径是否正确
   - 确认文件格式是否支持
   - PDF文件可能需要包含可提取的文本

4. **内容提取为空**
   - 网页可能使用了JavaScript动态加载内容
   - 文件可能为空或格式不支持
   - 尝试其他文件进行测试

5. **PPT转换失败**
   - 确保已安装Python依赖：`./install_python_deps.sh`
   - 检查Python版本是否为3.7+
   - 确保convert_to_ppt.py文件存在且有执行权限

## 测试

运行文件处理测试：

```bash
./test_file_processing.sh
```

这将创建一个测试TXT文件并演示文件处理功能。

## 许可证

MIT License

## 贡献

欢迎提交Issue和Pull Request来改进这个项目。 
package model

//配置模型

type PPTConfig struct {
	DefaultTemplate string `json:"default_template"` //默认模版名称
	DefaultTheme    string `json:"default_theme"`    //默认主题名称

	//页面
	PageWidth  int `json:"page_width"`  //页面宽度
	PageHeight int `json:"page_height"` //页面高度

	//字体配置
	DefaultFontFamily string `json:"default_font_family"` //默认字体
	DefaultFontSize   int    `json:"default_font_size"`   //默认字号

}

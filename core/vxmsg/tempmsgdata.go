package vxmsg

// NewTemplateMsg 创建一个新的模板消息对象（支持小程序跳转）
func NewTemplateMsg(toUser, templateID, url string, data map[string]interface{}, miniProgram *MiniProgram) TemplateMsg {
	return TemplateMsg{
		ToUser:      toUser,
		TemplateID:  templateID,
		URL:         url,
		Data:        data,
		MiniProgram: miniProgram, // 支持 nil，omitempty 会自动省略
	}
}

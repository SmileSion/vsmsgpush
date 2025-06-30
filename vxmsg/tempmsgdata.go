package vxmsg

// NewTemplateMsg 创建一个新的模板消息对象
func NewTemplateMsg(toUser, templateID, url string, data map[string]interface{}) TemplateMsg {
	return TemplateMsg{
		ToUser:     toUser,
		TemplateID: templateID,
		URL:        url,
		Data:       data,
	}
}

package models

type ClientContext struct {
	UserID          string
	Email           string
	Username        string
	DeviceID        string
	Architecture    string // Sec-CH-UA-Arch
	DeviceModel     string // Sec-CH-UA-Model
	Platform        string // Sec-CH-UA-Platform
	PlatformVersion string // Sec-CH-UA-Platform-Version
	UserAgent       string // Sec-CH-UA
	AppVersion      string // Sec-CH-UA-Full-Version
	IsMobile        string // Sec-CH-UA-Mobile
}

func (c *ClientContext) GetTargetingKey() string {
	if c.UserID != "" {
		return c.UserID
	}
	if c.DeviceID != "" {
		return c.DeviceID
	}
	return "anonymous"
}

func (c *ClientContext) IsAuthenticated() bool {
	return c.UserID != ""
}

package models

type ClientContext struct {
	AppVersion string
	Platform   string
	DeviceID   string
	UserID     string
	Email      string
	Username   string
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

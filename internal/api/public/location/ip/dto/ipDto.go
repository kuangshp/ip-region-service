package dto

type LookupRequest struct {
	IP string `json:"ip" form:"ip" binding:"required,max=64"`
}

package qianke

type QKUser struct {
	Email     string `json:"email"`
	Password  string `json:"passwd"`
	CellPhone string `json:"cellphone"`
	NickName  string `json:"nickname`
	Sex       int    `json:"sex"`
	Birth     string `json:"birth"`
	Job       string `json:"job"`
}

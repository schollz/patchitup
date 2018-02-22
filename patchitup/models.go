package patchitup

type serverRequest struct {
	Username string `json:"username" binding:"required"`
	Filename string `json:"filename" binding:"required"`
	Data     string `json:"data"`
}

type serverResponse struct {
	Message         string           `json:"message"`
	Success         bool             `json:"success"`
	HashLinenumbers map[string][]int `json:"hash_linenumbers"`
}

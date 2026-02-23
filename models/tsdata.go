package models

// TSData describes a monitoring item from proxtop
type TSData struct {
	Host    map[string]interface{}   `json:"host"`
	Domains []map[string]interface{} `json:"domains"`
}

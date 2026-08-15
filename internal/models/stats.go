package models

type QueueStats struct {
	Queued  int `json:"queued"`
	Running int `json:"running"`
	Done    int `json:"done"`
	Failed  int `json:"failed"`
}
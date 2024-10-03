package models

const (
	STATUS_QUEUED         = "queued"
	STATUS_RUNNING        = "running"
	STATUS_FAILED         = "failed"
	STATUS_COMPLETE       = "complete"
	STATUS_CONNECTING     = "connecting"
	STATUS_DISABLED       = "disabled"
	SCRIPTABLES_ROOT_PATH = "./scriptables"
	FLASH_SUCCESS         = "success"
	FLASH_ERROR           = "error"
)

type FlashMessage struct {
	Message     string
	MessageType string
}

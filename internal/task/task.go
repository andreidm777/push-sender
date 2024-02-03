package task

type Platform string

const (
	Android Platform = "android"
	Huawei  Platform = "huawei"
	Ios     Platform = "ios"
	Rustore Platform = "rustore"
)

type Task struct {
	ID      uint64
	Project string   `tnt:"0,require"`
	Type    Platform `tnt:"1,require"`
	To      string   `tnt:"2,require"`
	Payload any      `tnt:"3,require"`
}

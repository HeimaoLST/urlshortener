package util

type CustomError struct {
	Message string
	Code    string
}

func (e *CustomError) Error() string {
	return e.Message
}

var (
	ErrNotFoundInDB = &CustomError{Message: "短链接未找到", Code: "NOT_FOUND"}
	ErrDatabase     = &CustomError{Message: "数据库操作失败", Code: "DB_ERROR"}
)

package reconciliation

// StartAll 启动对账子系统的所有后台任务。
// main.go 仅需 import 本包并调用此函数即可，避免 main.go 出现多行启动调用，降低合并冲突面。
func StartAll() {
	StartBalanceTask()
	StartDailyTask()
}

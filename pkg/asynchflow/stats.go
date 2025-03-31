package asynchflow

import "fmt"

func (pc *AsynchFlow) GetStats(statsMap map[string]interface{}) error {

	// get session count using tls context
	sessionCount := pc.Ctx.GetConnectionCount()
	statsMap["sessionCount"] = fmt.Sprintf("%d", sessionCount)
	return nil
}

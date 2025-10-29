package socketasynch

import "fmt"

func (pc *SocketAsyncConnect) GetStats(statsMap map[string]interface{}) error {

	// get session count using session context
	sessionCount := pc.Ctx.GetConnectionCount()
	statsMap["sessionCount"] = fmt.Sprintf("%d", sessionCount)
	return nil
}

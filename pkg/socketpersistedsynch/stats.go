package socketpersistedsynch

import "fmt"

func (pc *SocketPersistedSyncConnect) GetStats(statsMap map[string]interface{}) error {

	// get session count using session context
	sessionCount := Ctx.GetConnectionCount()
	statsMap["sessionCount"] = fmt.Sprintf("%d", sessionCount)
	return nil
}

package https

import "fmt"

func (pc *HTTPPBMConnect) GetStats(statsMap map[string]interface{}) error {

	// get session count using tls context
	//sessionCount := Ctx.GetConnectionCount()
	statsMap["sessionCount"] = fmt.Sprintf("%d", 2)
	return nil
}

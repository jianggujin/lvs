package elevated

import "testing"

func TestName(t *testing.T) {
	err := SendPathUpdate("C:\\Users\\jianggujin\\.lvs\\symlink\\nodejs")
	if err != nil {
		t.Fatal(err)
	}
}

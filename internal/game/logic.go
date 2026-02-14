package game

func CheckWinner(b [9]string) (string, []int) {
	wins := [][]int{
		{0,1,2},{3,4,5},{6,7,8}, // Rows
		{0,3,6},{1,4,7},{2,5,8}, // Cols
		{0,4,8},{2,4,6},         // Diags
	}
	for _, w := range wins {
		if b[w[0]] != " " && b[w[0]] == b[w[1]] && b[w[1]] == b[w[2]] {
			return b[w[0]], w
		}
	}
	return "", nil
}

func CheckDraw(b [9]string) bool {
	for _, v := range b { if v == " " { return false } }
	return true
}

package app

// 根据令牌返回用户.
func getUserByToken(token string) member {
	// TODO: 根据 token 返回用户

	return getUserByCode("liuxue0905")
}

// 令牌生成.
func genToken(user *member) string {
	// TODO: 令牌生成

	return "liuxue0905_token"
}

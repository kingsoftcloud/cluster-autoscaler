package kce_client

func StringValue(v *string) string {
	if v != nil {
		return *v
	}
	return ""
}




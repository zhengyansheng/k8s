package kind

func toInt32(i int) int32 { return int32(i) }

func toInt64(i int) int64 { return int64(i) }

func int32Ptr(i int32) *int32 { return &i }

func int64Ptr(i int64) *int64 { return &i }

func stringPtr(i string) *string { return &i }

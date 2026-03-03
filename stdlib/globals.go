package stdlib

const GlobalsModule = `
func say(msg string) {}
func show(val any) {}
func inspect(val any) {}
func time() int { return 0 }
func date(format string, timestamp int) string { return "" }
func strtotime(datetime string) int { return 0 }
func sleep(seconds int) {}
func exit(code int) {}
func die(msg string) {}
func env(key string, default_val string) string { return "" }

func session_begin() {}
func session_set(key string, val string) {}
func session_get(key string) string { return "" }
func session_end() {}
`

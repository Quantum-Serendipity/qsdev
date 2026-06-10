package hardening

func Datamark(input string) string {
	return "[QSDEV:BEGIN]" + input + "[QSDEV:END]"
}

// Command noop does nothing. perfbudget's calibration times its start-up,
// the cost every Go process pays before running any of its own code, so the
// calibration factor tracks the machine and not qsdev: a qsdev start-up
// regression raises the measured rows but not the factor that scales their
// budgets. Keep it empty; the reference CPU in budgets.yaml was measured on
// this program.
package main

func main() {}

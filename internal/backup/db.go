package backup

import "fmt"

// A newer server is not refused, since nothing in the schema depends on what
// the versions above this one changed.
const MinimumPGVersion = 140000

func Describe(num int) string {
	if num == 0 {
		return "unknown"
	}
	return fmt.Sprintf("%d.%d", num/10000, num%10000)
}

// TooOld is what a server below MinimumPGVersion gets told, in the words of
// somebody who has never heard of a server_version_num.
func TooOld(found int) error {
	return fmt.Errorf(
		"this database is PostgreSQL %s, and pwikit needs %s or newer.\n"+
			"  Upgrade PostgreSQL, or point pwikit at a newer server with -database.\n"+
			"  To move the data: back it up with the pwikit you are running now\n"+
			"  (pwikit backup create), then restore it into the new server\n"+
			"  (pwikit backup restore <file> -database <new server>).",
		Describe(found), Describe(MinimumPGVersion))
}

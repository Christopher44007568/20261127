package main

// shortJointName returns compact uppercase labels that fit the heatmap row.
func shortJointName(j int) string {
	names := []string{
		"ROOT", "PELV", "SPIN", "CHST", "NECK", "HEAD",
		"LSHD", "LELB", "LHND",
		"RSHD", "RELB", "RHND",
		"LHIP", "LKNE", "LANK",
		"RHIP", "RKNE", "RANK",
	}
	return names[j]
}

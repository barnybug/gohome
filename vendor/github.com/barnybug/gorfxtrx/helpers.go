package gorfxtrx

func reverseByteStringMap(m map[byte]string) map[string]byte {
	ret := map[string]byte{}
	for k, v := range m {
		ret[v] = k
	}
	return ret
}

func decodeFlags(v byte, words []string) []string {
	s := []string{}
	for _, w := range words {
		if v%2 == 1 {
			s = append(s, w)
		}

		v = v / 2
	}
	return s
}

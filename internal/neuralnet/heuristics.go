package neuralnet

// heuristics.go contains helper functions for calculating contact input heuristics

// escapes1 is like Escapes but only counts escapes past the first blocked point
func escapes1(board [25]uint8, n int) int {
	initEscapeTables()

	af := 0
	m := n
	if m > 12 {
		m = 12
	}

	for i := 0; i < m; i++ {
		af |= anPoint[board[24+i-n]] << i
	}

	return anEscapes1[af]
}

// calculateTiming computes the timing heuristic
func calculateTiming(anBoard [25]uint8, nOppBack int) float32 {
	t := 0
	no := 0

	m := nOppBack
	if m < 11 {
		m = 11
	}

	t += 24 * int(anBoard[24])
	no += int(anBoard[24])

	for i := 23; i > m; i-- {
		nc := int(anBoard[i])
		if nc > 0 && nc != 2 {
			ns := nc - 2
			if ns < 1 {
				ns = 1
			}
			no += ns
			t += i * ns
		}
	}

	for i := m; i >= 6; i-- {
		nc := int(anBoard[i])
		if nc > 0 {
			no += nc
			t += i * nc
		}
	}

	for i := 5; i >= 0; i-- {
		nc := int(anBoard[i])
		if nc > 2 {
			t += i * (nc - 2)
			no += nc - 2
		} else if nc < 2 {
			nm := 2 - nc
			if no >= nm {
				t -= i * nm
				no -= nm
			}
		}
	}

	return float32(t) / 100.0
}

// calculateMoment computes the one-sided moment heuristic
func calculateMoment(anBoard [25]uint8) float32 {
	j := 0
	n := 0
	for i := 0; i < 25; i++ {
		ni := int(anBoard[i])
		if ni > 0 {
			j += ni
			n += i * ni
		}
	}

	if j == 0 {
		return 0.0
	}
	n = (n + j - 1) / j

	k := 0
	jCount := 0
	for i := n + 1; i < 25; i++ {
		ni := int(anBoard[i])
		if ni > 0 {
			jCount += ni
			k += ni * (i - n) * (i - n)
		}
	}

	if jCount > 0 {
		k = (k + jCount - 1) / jCount
	}

	return float32(k) / 400.0
}

// calculateBarEntry computes bar entry statistics
func calculateBarEntry(anBoard, anBoardOpp [25]uint8) (enter, enter2 float32) {
	if anBoard[24] > 0 {
		loss := 0
		two := anBoard[24] > 1

		for i := 0; i < 6; i++ {
			if anBoardOpp[i] > 1 {
				// Any double loses
				loss += 4 * (i + 1)

				for j := i + 1; j < 6; j++ {
					if anBoardOpp[j] > 1 {
						loss += 2 * (i + j + 2)
					} else if two {
						loss += 2 * (i + 1)
					}
				}
			} else if two {
				for j := i + 1; j < 6; j++ {
					if anBoardOpp[j] > 1 {
						loss += 2 * (j + 1)
					}
				}
			}
		}

		enter = float32(loss) / (36.0 * (49.0 / 6.0))
	}

	n := 0
	for i := 0; i < 6; i++ {
		if anBoardOpp[i] > 1 {
			n++
		}
	}
	enter2 = float32(36-(n-6)*(n-6)) / 36.0

	return enter, enter2
}

// calculateBackbone computes the backbone strength heuristic
func calculateBackbone(anBoard [25]uint8) float32 {
	ac := [23]int{11, 11, 11, 11, 11, 11, 11, 6, 5, 4, 3, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	pa := -1
	w := 0
	tot := 0

	for np := 23; np > 0; np-- {
		if anBoard[np] >= 2 {
			if pa == -1 {
				pa = np
				continue
			}
			d := pa - np
			w += ac[d] * int(anBoard[pa])
			tot += int(anBoard[pa])
		}
	}

	if tot > 0 {
		return 1.0 - float32(w)/float32(tot*11)
	}
	return 0.0
}

// calculateBackGame computes back game indicators
func calculateBackGame(anBoard [25]uint8) (backg, backg1 float32) {
	nAc := 0
	for i := 18; i < 24; i++ {
		if anBoard[i] > 1 {
			nAc++
		}
	}

	if nAc >= 1 {
		tot := 0
		for i := 18; i < 25; i++ {
			tot += int(anBoard[i])
		}

		if nAc > 1 {
			backg = float32(tot-3) / 4.0
		} else {
			backg1 = float32(tot) / 8.0
		}
	}

	return backg, backg1
}

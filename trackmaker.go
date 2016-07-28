package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fohristiwhirl/wavmaker"
)

var midi_freq [128]float64 = [128]float64{
	8.175799, 8.661957, 9.177024, 9.722718, 10.300861, 10.913382, 11.562326, 12.249857,
	12.978272, 13.750000, 14.567618, 15.433853, 16.351598, 17.323914, 18.354048, 19.445436,
	20.601722, 21.826764, 23.124651, 24.499715, 25.956544, 27.500000, 29.135235, 30.867706,
	32.703196, 34.647829, 36.708096, 38.890873, 41.203445, 43.653529, 46.249303, 48.999429,
	51.913087, 55.000000, 58.270470, 61.735413, 65.406391, 69.295658, 73.416192, 77.781746,
	82.406889, 87.307058, 92.498606, 97.998859, 103.826174, 110.000000, 116.540940, 123.470825,
	130.812783, 138.591315, 146.832384, 155.563492, 164.813778, 174.614116, 184.997211, 195.997718,
	207.652349, 220.000000, 233.081881, 246.941651, 261.625565, 277.182631, 293.664768, 311.126984,
	329.627557, 349.228231, 369.994423, 391.995436, 415.304698, 440.000000, 466.163762, 493.883301,
	523.251131, 554.365262, 587.329536, 622.253967, 659.255114, 698.456463, 739.988845, 783.990872,
	830.609395, 880.000000, 932.327523, 987.766603, 1046.502261, 1108.730524, 1174.659072, 1244.507935,
	1318.510228, 1396.912926, 1479.977691, 1567.981744, 1661.218790, 1760.000000, 1864.655046, 1975.533205,
	2093.004522, 2217.461048, 2349.318143, 2489.015870, 2637.020455, 2793.825851, 2959.955382, 3135.963488,
	3322.437581, 3520.000000, 3729.310092, 3951.066410, 4186.009045, 4434.922096, 4698.636287, 4978.031740,
	5274.040911, 5587.651703, 5919.910763, 6271.926976, 6644.875161, 7040.000000, 7458.620184, 7902.132820,
	8372.018090, 8869.844191, 9397.272573, 9956.063479, 10548.081821, 11175.303406, 11839.821527, 12543.853951,
}

type Instrument struct {
	notes [128]*wavmaker.WAV
	flags [128]bool
	ready bool
}

type ParserState struct {
	line uint32					// current line in score
	position uint32				// position in samples e.g. 44100 means 1 second in
	jump uint32
	instrument_name string
	volume float64
	drunk int32					// signed is correct since rand.Int31n() takes an int32 arg
	offset uint32
}

type Insertion struct {
	instrument_name string
	note_name string
	timing uint32				// "timing" is the position it would have if not for offsets caused by d: and o:
	timing_adjusted uint32		// whereas "timing_adjusted" is the actual position in the target WAV.
	volume float64
}

var instruments = make(map[string]*Instrument)
var default_instrument_name string


// ---------------------------------------------------------- METHODS


func (instrument *Instrument) addfile(notestring string, filename string) error {

	note, err := name_to_midi(notestring)
	if err != nil {
		return err
	}

	wav, err := wavmaker.Load(filename)

	if err != nil {
		return err
	}

	instrument.notes[note] = wav
	instrument.flags[note] = true
	instrument.ready = true

	return nil
}


// ---------------------------------------------------------- FUNCTIONS


func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}


func main() {

	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s directory\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}
	err := os.Chdir(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	load_instruments("instruments.txt")
	score_to_wav("score.txt", "trackmaker_output.wav")
}


func load_instruments(filename string) {

	var scanner *bufio.Scanner

	instruments_file, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't read %s\n", filename)
		os.Exit(1)
	}
	defer instruments_file.Close()

	scanner = bufio.NewScanner(instruments_file)

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 3 {

			insname, notename, filename := fields[0], fields[1], fields[2]

			// Format is:    piano G4 piano.ff.G4.wav

			if default_instrument_name == "" {
				default_instrument_name = insname
			}

			ins, ok := instruments[insname]
			if ok == false {
				ins = new(Instrument)
				instruments[insname] = ins
			}

			err = ins.addfile(notename, filename)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Couldn't add %s to %s: %v\n", filename, insname, err)
			}
		}
	}
}


func score_to_wav(filename string, outfilename string) {

	score_file, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't read %s\n", filename)
		os.Exit(1)
	}
	defer score_file.Close()

	// Parsing phase...

	parser_state := initial_parser_state()
	scanner := bufio.NewScanner(score_file)

	all_inserts := make([]Insertion, 0)

	for scanner.Scan() {
		new_inserts := handle_score_line(&parser_state, scanner.Text())
		all_inserts = append(all_inserts, new_inserts...)
	}

	// WAV creation phase...

	z := len(all_inserts)
	if z == 0 {
		fmt.Fprintf(os.Stderr, "No valid inserts seen in the score.\n")
		os.Exit(1)
	}

	output := wavmaker.New(all_inserts[z - 1].timing + 44100 * 5)		// 5 seconds grace period at end

	for _, insert := range all_inserts {
		err := insert_by_name(insert.instrument_name, insert.volume, insert.note_name, output, insert.timing_adjusted)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n")
		}
	}

	output.FadeSamples(44100)
	output.Save(outfilename)
}


func initial_parser_state() ParserState {		// Set all things that need to be non-zero
	var s = ParserState{
			instrument_name : default_instrument_name,
			volume : 1.0,
			jump : 11025,
	}
	return s
}


func handle_score_line(global_state *ParserState, text string) []Insertion {

	// This function uses the parser state to handle a line of text, adjusting the
	// parser state and also returning all new inserts (i.e. notes) found.

	new_inserts := make([]Insertion, 0)

	// Since brackets are significant, make sure they are isolated for convenience...

	text = strings.Replace(text, "(", " ( ", -1)
	text = strings.Replace(text, ")", " ) ", -1)

	fields := strings.Fields(text)

	// When we're inside brackets, we don't change the global state but just a local one.
	// The pointer "relevant_state" will point to the one currently being updated. Note that
	// each time we enter brackets, the global state must be copied into the local state.
	// Only one layer of brackets is allowed (no nesting).

	var local_state *ParserState = new(ParserState)

	var relevant_state *ParserState = global_state

	for _, token := range fields {

		// Deal with brackets first...

		if token == "(" && relevant_state == global_state {
			*local_state = *global_state	// Copy contents
			relevant_state = local_state
			continue
		}

		if token == ")" && relevant_state == local_state {
			relevant_state = global_state
			continue
		}

		// Branch based on whether the token is a note...

		_, err := name_to_midi(token)	// FIXME: using this function just for its err is crude
		if err != nil {

			// instrument name? ---------------------------------------------------------------- e.g. piano

			_, ok := instruments[token]
			if ok {
				relevant_state.instrument_name = token
				continue
			}

			// jump setting? (i.e. frames between notes) --------------------------------------- e.g. j:5000

			if strings.HasPrefix(token, "j:") {
				j, err := strconv.Atoi(token[2:])
				if err != nil {
					fmt.Fprintf(os.Stderr, "line %d: bad token \"%s\"\n", relevant_state.line, token)
				} else {
					relevant_state.jump = uint32(j)
				}
				continue
			}

			// offset setting ------------------------------------------------------------------ e.g. o:2000

			if strings.HasPrefix(token, "o:") {
				o, err := strconv.Atoi(token[2:])
				if err != nil {
					fmt.Fprintf(os.Stderr, "line %d: bad token \"%s\"\n", relevant_state.line, token)
				} else {
					relevant_state.offset = uint32(o)
				}
				continue
			}

			// drunk setting? (random delay before playing a note) ----------------------------- e.g. d:300

			if strings.HasPrefix(token, "d:") {
				d, err := strconv.Atoi(token[2:])
				if err != nil {
					fmt.Fprintf(os.Stderr, "line %d: bad token \"%s\"\n", relevant_state.line, token)
				} else {
					relevant_state.drunk = int32(d)
				}
				continue
			}

			// volume setting? (as a float where 1.0 means normal) ----------------------------- e.g. v:0.5

			if strings.HasPrefix(token, "v:") {
				v, err := strconv.ParseFloat(token[2:], 64)
				if err != nil {
					fmt.Fprintf(os.Stderr, "line %d: bad token \"%s\"\n", relevant_state.line, token)
				} else {
					relevant_state.volume = v
				}
				continue
			}

			// We didn't figure out what the token means ---------------------------------------

			fmt.Fprintf(os.Stderr, "line %d: unknown token \"%s\"\n", relevant_state.line, token)

		} else {

			// The token is a note...

			new_inserts = append(new_inserts, Insertion{
					instrument_name : relevant_state.instrument_name,
					note_name : token,
					timing : relevant_state.position,
					timing_adjusted : relevant_state.position + uint32(safe_int31n(relevant_state.drunk)) + relevant_state.offset,
					volume : relevant_state.volume,
				})
		}
	}

	global_state.line += 1
	global_state.position += global_state.jump

	return new_inserts
}


func safe_int31n(n int32) int32 {
	if n <= 0 {
		return 0
	}
	return rand.Int31n(n)
}


func insert_by_name(instrument_name string, volume float64, note_name string, target_wav *wavmaker.WAV, t_loc uint32) error {

	// Get the named instrument from the global instruments map,
	// and insert it into the wav with the given note, creating
	// that note if needed...

	i, ok := instruments[instrument_name]
	if ok == false {
		return fmt.Errorf("insert_by_name() couldn't find instrument \"%s\"", instrument_name)
	}

	if i.ready == false {
		return fmt.Errorf("insert_by_name() called on an empty instrument")
	}

	note, err := name_to_midi(note_name)		// A number between 0 and 127 (MIDI value corresponding to note)
	if err != nil {
		return fmt.Errorf("insert_by_name(): %v", err)
	}

	if i.notes[note] == nil {

		a := int(note)
		b := int(note)

		note_to_stretch := 0

		for {		// Find reference note (one with its flag set) which was loaded from a file
			a--
			b++

			if a >= 0 {
				if i.flags[a] {
					note_to_stretch = a
					break
				}
			}

			if b <= 127 {
				if i.flags[b] {
					note_to_stretch = b
					break
				}
			}

			if a <= 0 && b >= 127 {
				return fmt.Errorf("insert() couldn't find a reference note")	// Should be impossible
			}
		}

		ins_freq := midi_freq[note]
		ref_freq := midi_freq[note_to_stretch]

		i.notes[note] = i.notes[note_to_stretch].StretchedRelative(ref_freq / ins_freq)
	}

	target_wav.Add(t_loc, i.notes[note], 0, i.notes[note].FrameCount(), volume, 0)
	return nil
}


func name_to_midi(name string) (int, error) {

	// Accepts notes in the following formats: C4  C4#  C#4  C4b  Cb4

	var result, number, accidental int
	var letter string

	if len(name) == 2 {
		letter = string(name[0])
		letter = strings.ToUpper(letter)
		number = int(name[1]) - 48				// -48 is conversion of ASCII to int
	} else if len(name) == 3 {
		letter = string(name[0])
		letter = strings.ToUpper(letter)
		if name[1] == '#' || name[1] == 'b' {
			number = int(name[2]) - 48
			if name[1] == '#' {
				accidental = 1
			} else {
				accidental = -1
			}
		} else if name[2] == '#' || name[2] == 'b' {
			number = int(name[1]) - 48
			if name[2] == '#' {
				accidental = 1
			} else {
				accidental = -1
			}
		} else {
			return 0, fmt.Errorf("name_to_midi(%s): string format was wrong", name)
		}
	} else {
		return 0, fmt.Errorf("name_to_midi(%s): string length was wrong", name)
	}

	// First we set the result as if we asked for C in the relevant octave...

	switch number {
		case 0: result = 12		// C0
		case 1: result = 24		// C1
		case 2:	result = 36		// C2
		case 3:	result = 48		// C3
		case 4:	result = 60		// C4
		case 5:	result = 72		// C5
		case 6:	result = 84		// C6
		case 7:	result = 96		// C7
		case 8:	result = 108	// C8
		case 9: result = 120	// C9
		default: return 0, fmt.Errorf("name_to_midi(%s): note number was wrong", name)
	}

	// Now we adjust it for the actual note that was requested...

	switch letter {
		case "C": result += 0
		case "D": result += 2
		case "E": result += 4
		case "F": result += 5
		case "G": result += 7
		case "A": result += 9
		case "B": result += 11
		default: return 0, fmt.Errorf("name_to_midi(%s): note letter was wrong", name)
	}

	// Now take into account flat or sharp symbols...

	result += accidental

	if result < 0 || result > 127 {
		return 0, fmt.Errorf("name_to_midi(%s): resulting note out of range 0-127", name)
	}

	return result, nil
}

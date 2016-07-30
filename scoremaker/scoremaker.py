# This is a crude editor I wrote two years ago
# with a few minor edits in the present day
#
# Keys:
#
#	SAVE: Space
#	SCROLL: Up and Down arrows
#	ADD / REMOVE note: mouseclick, S (sharp), F (flat)


import pygame, sys
from pygame.locals import *

INPUT_FILENAME = "score.txt"
OUTPUT_FILENAME = "score.txt"

FILE_START = ""

WINDOW_WIDTH = 800
WINDOW_HEIGHT = 420

NOTES_PER_BAR = 8
BARS_PER_STAFF = 4

LINE_GAP = 16			# best make this even
X_MARGIN = 24

NOTE_WIDTH = 16
NOTE_HEIGHT = 16

bar_width = WINDOW_WIDTH // BARS_PER_STAFF

note_x_pixel_locations = []
x = X_MARGIN
while 1:
	note_x_pixel_locations.append(round(x))
	if len(note_x_pixel_locations) == NOTES_PER_BAR:
		break
	x += ((bar_width - X_MARGIN) - X_MARGIN) / (NOTES_PER_BAR - 1)

note_from_distance = {
		-20 : "B6", -19 : "A6", -18 : "G6", -17 : "F6", -16 : "E6",
		-15 : "D6", -14 : "C6", -13 : "B5", -12 : "A5", -11 : "G5",
		-10 : "F5", -9 : "E5", -8 : "D5", -7 : "C5", -6 : "B4",
		 -5 : "A4", -4 : "G4", -3 : "F4", -2 : "E4", -1 : "D4",
		  0 : "C4",
		  1 : "B3", 2 : "A3", 3 : "G3", 4 : "F3", 5 : "E3",
		  6 : "D3", 7 : "C3", 8 : "B2", 9 : "A2", 10 : "G2",
		 11 : "F2", 12 : "E2", 13 : "D2", 14 : "C2", 15 : "B1",
		 16 : "A1", 17 : "G1", 18 : "F1", 19 : "E1", 20 : "D1"
	}

legal_inputs = []
for letter in "CDEFGAB":
	for number in "012345678":
		legal_inputs.append(letter + number)
		for accidental in ["b", "#"]:
			legal_inputs.append(letter + number + accidental)
			legal_inputs.append(letter + accidental + number)

#-------------------------------------------------------------------------------------------------------

class Bar (object):

	def __init__(self, notes_per_bar):
		self.locations = [set() for x in range(notes_per_bar)]
		self.notes_per_bar = notes_per_bar

	def handle_click(self, x, y, accidental=""):
		y_distance = y - WINDOW_HEIGHT // 2
		letter_distance = round(y_distance / (LINE_GAP // 2))
		try:
			note = note_from_distance[letter_distance]
		except KeyError:
			return
		if accidental:
			assert(accidental in "#b")
			note += accidental
		note = name_to_canonical_name(note)
		if note == "":
			return
		location = closest_member_index(note_x_pixel_locations, x)
		if note in self.locations[location]:
		 	self.locations[location].discard(note)
		else:
			self.locations[location].add(note)

	def draw(self, target, x_offset):
		for time, note_list in enumerate(self.locations):
			for note in note_list:
				x = note_x_pixel_locations[time] + x_offset
				try:
					y = get_distance_from_note(note) * LINE_GAP // 2 + WINDOW_HEIGHT // 2
					if "#" in note:
						blit(target, sharpSprite, x, y)
					elif "b" in note:
						blit(target, flatSprite, x, y)
					else:
						blit(target, noteSprite, x, y)
				except AssertionError:
					pass

class Staff (object):

	def __init__(self):
		self.bars = [Bar(NOTES_PER_BAR) for x in range(BARS_PER_STAFF)]

	def draw(self, target):
		C4_y = WINDOW_HEIGHT // 2
		for j in range(LINE_GAP, LINE_GAP * 9, LINE_GAP):
			if j <= LINE_GAP * 5:
				colour = pygame.Color(0,0,0)
			else:
				colour = pygame.Color(220,220,220)
			pygame.draw.line(target, colour, (0, C4_y + j), (WINDOW_WIDTH, C4_y + j))
			pygame.draw.line(target, colour, (0, C4_y - j), (WINDOW_WIDTH, C4_y - j))

		pygame.draw.line(target, pygame.Color(220,220,220), (0, C4_y), (WINDOW_WIDTH, C4_y))

		for i in range(1, BARS_PER_STAFF):
			pygame.draw.line(target, pygame.Color(0,0,0),
							(i * WINDOW_WIDTH // BARS_PER_STAFF, C4_y - LINE_GAP * 5),
							(i * WINDOW_WIDTH // BARS_PER_STAFF, C4_y + LINE_GAP * 5)
							)

		for i, bar in enumerate(self.bars):
			bar.draw(target, i * bar_width)

	def handle_click(self, x, y, accidental=""):
		bar = x // (WINDOW_WIDTH // BARS_PER_STAFF)
		if bar >= len(self.bars):
			bar = len(self.bars) - 1
		x -= bar * WINDOW_WIDTH // BARS_PER_STAFF
		self.bars[bar].handle_click(x, y, accidental)

class Score (object):

	def __init__(self, filename=None):
		self.staffs = [Staff()]
		self.__displayed_staff = 0
		if filename is not None:
			self.load(filename)

	@property
	def displayed_staff(self):
		return self.__displayed_staff

	@displayed_staff.setter
	def displayed_staff(self, displayed_staff):
		if displayed_staff < 0:
			displayed_staff = 0
		self.__displayed_staff = displayed_staff
		if len(self.staffs) <= displayed_staff:
			for i in range(len(self.staffs), displayed_staff + 1):
				self.staffs.append(Staff())

	def draw(self, target):
		target.fill(pygame.Color(255,255,255))
		self.staffs[self.displayed_staff].draw(target)

	def handle_click(self, x, y, accidental=""):
		self.staffs[self.displayed_staff].handle_click(x, y, accidental)

	def title(self):
		return "Staff " + str(self.displayed_staff) + " of [0 to " + str(len(self.staffs) - 1) + "]"

	def load(self, filename):		# Can be used to merge input files since it doesn't delete notes already present
		try:
			infile = open(filename)
		except FileNotFoundError:
			return
		staff_n = 0
		bar_n = 0
		notes_loc = 0
		for t, line in enumerate(infile):
			line = line.replace("(", "")
			line = line.replace(")", "")
			line = line.split()
			for token in line:
				if token in legal_inputs:
					self.staffs[staff_n].bars[bar_n].locations[notes_loc].add(name_to_canonical_name(token))
			notes_loc += 1
			if notes_loc >= NOTES_PER_BAR:
				notes_loc = 0
				bar_n += 1
				if bar_n >= BARS_PER_STAFF:
					bar_n = 0
					staff_n += 1
					if len(self.staffs) == staff_n:
						self.staffs.append(Staff())
		infile.close()

	def save(self, filename):
		outfile = open(filename, "w")
		outfile.write(FILE_START + " ")
		for staff in self.staffs:
			for bar in staff.bars:
				for loc in bar.locations:
					for note in loc:
						outfile.write(note)
						outfile.write(" ")
					outfile.write("\n")
		outfile.close()


#-------------------------------------------------------------------------------------------------------

def get_distance_from_note(note):
	if note[1] in "#b":
		note = note[0] + note[2]
	for key,val in note_from_distance.items():
		if val == note[0:2]:
			return key
	assert(1 == 0)

def handle_events():
	global mousex; global mousey
	global mouseclickx; global mouseclicky
	global keyboard

	mouseclickx, mouseclicky = None, None

	for event in pygame.event.get():
		if event.type == QUIT:
			pygame.quit()
			sys.exit()
		if event.type == MOUSEMOTION:
			mousex, mousey = event.pos
		if event.type == MOUSEBUTTONDOWN:
			mouseclickx, mouseclicky = mousex, mousey
		if event.type == KEYDOWN:
			keyboard[event.key] = 1
		if event.type == KEYUP:
			keyboard[event.key] = 0

def startup():

	pygame.init()

	global mousex; global mousey
	mousex, mousey = 0, 0

	global keyboard
	keyboard = dict()

	global virtue; virtue = pygame.display.set_mode((WINDOW_WIDTH, WINDOW_HEIGHT))
	virtue.fill(pygame.Color(255,255,255))
	pygame.display.update()

	global noteSprite; noteSprite = pygame.image.load("note.png").convert_alpha()
	global sharpSprite; sharpSprite = pygame.image.load("sharp.png").convert_alpha()
	global flatSprite; flatSprite = pygame.image.load("flat.png").convert_alpha()

	global fpsClock; fpsClock = pygame.time.Clock()

def closest_member_index(iterable, val):	# What index in the list has the
	best_diff = None						# value closest to the passed value?
	best_index = None
	for i, n in enumerate(iterable):
		diff = abs(n - val)
		if best_diff is None:
			best_diff = diff
			best_index = i
		if diff < best_diff:
			best_diff = diff
			best_index = i
	return best_index

def name_to_canonical_name(name):
	if len(name) not in [2,3]:
		return ""
	if name[0] not in "CDEFGAB":
		return ""
	if name[1] not in "#b012345678":
		return ""
	if len(name) == 2:
		if name[1] not in "012345678":
			return ""
	if len(name) == 3:
		if name[2] not in "#b012345678":
			return ""
		if name[1] in "#b":
			if name[2] not in "012345678":
				return ""
		if name[2] in "#b":
			if name[1] not in "012345678":
				return ""

	# Name is now guaranteed to be in one of these formats: C4 C4# C4b C#4 Cb4

	letter = name[0]
	if len(name) == 2:
		number = int(name[1])
		accidental = ""
	else:
		if name[1] in "012345678":
			number = int(name[1])
			accidental = name[2]
		else:
			number = int(name[2])
			accidental = name[1]

	# Deal with the 4 equivalences:

	if letter == "B" and accidental == "#":
		letter = "C"
		number += 1
		accidental = ""
	if letter == "C" and accidental == "b":
		letter = "B"
		number -= 1
		accidental = ""
	if letter == "E" and accidental == "#":
		letter = "F"
		accidental = ""
	if letter == "F" and accidental == "b":
		letter = "E"
		accidental = ""

	return letter + accidental + str(number)

def blit(target, source, x, y):
	topleftx = int(x - source.get_width() // 2)
	toplefty = int(y - source.get_height() // 2)
	target.blit(source, (topleftx, toplefty))

def main():
	startup()
	score = Score(INPUT_FILENAME)

	while 1:
		handle_events()
		if mouseclickx and mouseclicky:
			score.handle_click(mouseclickx, mouseclicky)
		if keyboard.setdefault(K_DOWN, 0):
			keyboard[K_DOWN] = 0
			score.displayed_staff += 1
		if keyboard.setdefault(K_UP, 0):
			keyboard[K_UP] = 0
			score.displayed_staff -= 1
		if keyboard.setdefault(K_s, 0):
			keyboard[K_s] = 0
			score.handle_click(mousex, mousey, accidental="#")
		if keyboard.setdefault(K_f, 0):
			keyboard[K_f] = 0
			score.handle_click(mousex, mousey, accidental="b")
		if keyboard.setdefault(K_SPACE, 0):
			keyboard[K_SPACE] = 0
			score.save(OUTPUT_FILENAME)

		score.draw(virtue)
		pygame.display.set_caption(score.title())
		pygame.display.update()
		fpsClock.tick(85)

main()

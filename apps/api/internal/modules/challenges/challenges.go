// Package challenges owns short, guided, multi-day experiences a couple
// takes on together. Each partner marks their own days (DEC-30); skipping is
// a first-class answer, not a penalty (01 §6).
package challenges

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

var (
	ErrNotFound = apperr.NotFound("challenge_not_found", "You don’t have a challenge going.")
	// The same curated challenge twice at once; one they wrote can be repeated.
	ErrAlreadyRunning = apperr.Conflict("challenge_already_running", "That one’s already running.")
	// MaxActive going at once.
	ErrTooMany         = apperr.Conflict("too_many_challenges", "Three at once is plenty — finish or end one first.")
	ErrUnknownTemplate = apperr.Invalid("challenge_unknown", "That challenge isn’t one of ours.")
	ErrUnknownDay      = apperr.NotFound("challenge_day_not_found", "That day isn’t part of this challenge.")
	// Days open one at a time, on the couple's own calendar.
	ErrDayNotOpen = apperr.Conflict("day_not_open", "That day isn’t here yet.")
	// Finished or left: kept to look back on, no longer to be marked.
	ErrOver = apperr.Conflict("challenge_over", "This one’s finished.")
	// A reflection is for looking back, so it waits for there to be a back.
	ErrNotOver = apperr.Conflict("challenge_not_over", "Save that for when this one is over.")
	// A past challenge asked for by id that is not this couple's, or not anyone's.
	ErrNoSuchChallenge = apperr.NotFound("challenge_not_found", "We can’t find that challenge.")
	// The start can move only while nobody has begun.
	ErrBegun = apperr.Conflict("challenge_begun", "It’s already begun — the start can’t move now.")
	// Together to just-me only while the other partner has not joined in.
	ErrKindLocked = apperr.Conflict("challenge_kind_locked", "Your partner has already joined in.")
)

// ErrDayHasMarks is the refusal to drop a day somebody has marked or written
// on; it names the day so the screen can say which.
func ErrDayHasMarks(n int) error {
	return apperr.Conflict("day_has_marks", fmt.Sprintf("Day %d has been marked, so it can’t be removed.", n))
}

// ErrPartnerFull is the cap hit on the other person's side: a together
// challenge counts for both of them.
func ErrPartnerFull(name string) error {
	if name == "" {
		return apperr.Conflict("too_many_challenges", "One of you already has three going.")
	}
	return apperr.Conflict("too_many_challenges", name+" already has three going.")
}

// Status is where a challenge is up to. A couple can have several active, up
// to MaxActive.
type Status string

const (
	StatusActive   Status = "active"
	StatusFinished Status = "finished"
	StatusEnded    Status = "ended"
)

// Kind is whose a challenge is. "together" is for both of them; "mine" is one
// person's own — the other can read it, not touch it.
type Kind string

const (
	KindTogether Kind = "together"
	KindMine     Kind = "mine"
)

// MaxActive is how many challenges a person can have going at once, counting
// the ones they share and their own "just me" ones (not their partner's).
// More than this is not a practice, it is a pile; the limit is enforced when
// one is started, not by the schema.
const MaxActive = 3

// MaxStartAhead is how far ahead, in days, a challenge may be scheduled.
const MaxStartAhead = 60

const (
	MaxNoteRunes       = 280
	MaxReflectionRunes = 1000
	MaxCustomTitle     = 80
	MaxCustomPrompt    = 200
	MinCustomDays      = 3
	// A hundred, not forty: a habit (thirty days off soda, ninety of
	// walking) is one line repeated, and the client sends it that way.
	MaxCustomDays = 100
)

// Mark is what one partner has said about one day.
type Mark string

const (
	MarkDone    Mark = "done"
	MarkSkipped Mark = "skipped"
)

type Challenge struct {
	ID       uuid.UUID
	CoupleID uuid.UUID
	Template string
	Title    string
	Status   Status
	Kind     Kind
	// Dates are couple-local calendar days held at midnight UTC, so
	// subtracting one from another counts days with no timezone in the way.
	StartedOn time.Time
	// Today is the couple's own date when this was read. The repository
	// takes it from the database, which knows their zone, so the two of them
	// always agree on it.
	Today   time.Time
	EndedAt *time.Time
	// Who started it; the zero id once that person is gone.
	CreatedBy uuid.UUID
	Days      []Day
	// What each of them wrote once it was over.
	Reflections map[uuid.UUID]string
}

// TodayN is the day the couple is on: days since it started, counting from
// one, held at the last. Zero before it has begun: there is no day yet.
func (c Challenge) TodayN() int {
	n := int(c.Today.Sub(c.StartedOn)/(24*time.Hour)) + 1
	if n > len(c.Days) {
		n = len(c.Days)
	}
	if n < 1 {
		n = 0
	}
	return n
}

// Begun is whether the start day has come.
func (c Challenge) Begun() bool { return !c.StartedOn.After(c.Today) }

// StartsIn is how many days until it begins; zero once it has.
func (c Challenge) StartsIn() int {
	if c.Begun() {
		return 0
	}
	return int(c.StartedOn.Sub(c.Today) / (24 * time.Hour))
}

// MayTouch is whether this person can write to it: either partner for a
// shared one, only whoever started it for a "just me" one.
func (c Challenge) MayTouch(userID uuid.UUID) bool {
	return c.Kind != KindMine || (c.CreatedBy != uuid.Nil && c.CreatedBy == userID)
}

// anyoneElseMarked is whether somebody other than this person has a mark or
// a note on any day.
func (c Challenge) anyoneElseMarked(userID uuid.UUID) bool {
	for _, d := range c.Days {
		for u := range d.Marks {
			if u != userID {
				return true
			}
		}
		for u := range d.Notes {
			if u != userID {
				return true
			}
		}
	}
	return false
}

// DateOf is the calendar day n opens on — the same day for both of them.
func (c Challenge) DateOf(n int) time.Time { return c.StartedOn.AddDate(0, 0, n-1) }

// Opened is whether day n has come yet. Earlier days stay open, so a day
// missed can still be caught up on.
func (c Challenge) Opened(n int) bool { return !c.DateOf(n).After(c.Today) }

// Summary is a finished or left challenge as the list of past ones shows it.
type Summary struct {
	ID          uuid.UUID
	Template    string
	Title       string
	Status      Status
	Kind        Kind
	CreatedBy   uuid.UUID
	StartedOn   time.Time
	EndedAt     *time.Time
	Days        int
	MyDone      int
	PartnerDone int
}

type Day struct {
	ID     uuid.UUID
	N      int
	Prompt string
	// Absent means unmarked, distinct from skipped.
	Marks map[uuid.UUID]Mark
	// Each person's own words on the day; absent means none.
	Notes map[uuid.UUID]string
}

func (d Day) MarkFor(userID uuid.UUID) (Mark, bool) {
	m, ok := d.Marks[userID]
	return m, ok
}

// Template is a curated challenge. Kept in code, not a DB table — these read
// like copy and a typo fix shouldn't need a migration.
type Template struct {
	Key   string
	Title string
	Blurb string
	// "" for one that can be started any time, or the season it belongs to.
	Season string
	// One of the Category constants; how the library is grouped.
	Category string
	Prompts  []string
}

const (
	SeasonAdvent = "advent"
	SeasonLent   = "lent"

	CategoryConnection = "connection"
	CategoryFaith      = "faith"
	CategoryService    = "service"
	CategorySeason     = "season"

	// What a challenge the couple wrote themselves is filed under.
	CustomKey = "custom"
)

var templates = []Template{
	{
		Key:      "seven-days-of-noticing",
		Title:    "Seven days of noticing",
		Blurb:    "One small thing a day, about each other.",
		Category: CategoryConnection,
		Prompts: []string{
			"Tell them one thing they did this week that you noticed.",
			"Ask about something they are looking forward to.",
			"Say thank you for something ordinary.",
			"Put your phones down for one meal together.",
			"Tell them something you admire that they would not guess.",
			"Ask what would make tomorrow easier for them, and do it.",
			"Say what this week has been like for you, honestly.",
		},
	},
	{
		Key:      "seven-days-of-praying-together",
		Title:    "Seven days of praying together",
		Blurb:    "A short prayer each day, out loud, together.",
		Category: CategoryFaith,
		Prompts: []string{
			"Pray for each other's week ahead.",
			"Pray for your families.",
			"Pray about something one of you is worried about.",
			"Give thanks for three things from this year.",
			"Pray for someone outside the two of you.",
			"Pray about your home, wherever it is.",
			"Pray for the two of you, five years from now.",
		},
	},
	{
		Key:      "fourteen-days-of-small-things",
		Title:    "Fourteen days of small things",
		Blurb:    "Two weeks of very small, very doable things.",
		Category: CategoryConnection,
		Prompts: []string{
			"Make them a drink the way they like it.",
			"Send a message in the middle of the day for no reason.",
			"Take one chore off their list without mentioning it.",
			"Ask about their day and let them finish.",
			"Share a memory from before you lived together.",
			"Go to bed at the same time.",
			"Tell them one thing you are proud of them for.",
			"Plan something small for next weekend.",
			"Sit outside together for ten minutes.",
			"Ask what they need this week.",
			"Say sorry for something small you never said sorry for.",
			"Look at old photos together.",
			"Do nothing together, on purpose, for half an hour.",
			"Tell them what these two weeks have been like.",
		},
	},
	{
		Key:      "seven-days-of-gratitude",
		Title:    "Seven days of gratitude",
		Blurb:    "One thing a day to be thankful for, said out loud.",
		Category: CategoryConnection,
		Prompts: []string{
			"Tell them one thing about them you are thankful for today.",
			"Name something in your home you are glad to have. “Give thanks in every thing” (1 Thessalonians 5:18).",
			"Thank God together for a meal, and for whoever made it possible.",
			"Say thank you for something they do that they think goes unnoticed.",
			"Share a hard thing that turned out to be a gift, looking back.",
			"Thank someone else together: a message, a call, or a note.",
			"Say what you are most thankful for about this week, and about each other.",
		},
	},
	{
		Key:      "seven-days-of-serving-each-other",
		Title:    "Seven days of serving each other",
		Blurb:    "A quiet act of service a day, for the other one.",
		Category: CategoryService,
		Prompts: []string{
			"Ask what one thing you could take off their plate today, and do it.",
			"Do something for them that they usually do for themselves.",
			"Put their needs first at one decision today, even a small one.",
			"Prepare something for them without being asked: a meal, a bath, a bag packed.",
			"Listen for ten minutes without fixing anything. “By love serve one another” (Galatians 5:13).",
			"Pray for them by name, then do one thing that answers your own prayer.",
			"Ask how you served them well this week, and how you could serve them better.",
		},
	},
	{
		Key:      "fourteen-days-in-the-psalms",
		Title:    "Fourteen days in the Psalms",
		Blurb:    "One psalm a day, and one line to share.",
		Category: CategoryFaith,
		Prompts: []string{
			"Read Psalm 1 together. Share the line you would like to be true of your home.",
			"Read Psalm 8 together. Share what makes you feel small in a good way.",
			"Read Psalm 19 together. Share something you have seen lately that pointed to God.",
			"Read Psalm 23 together. Share the line that feels most like where you are right now.",
			"Read Psalm 27 together. Share one thing you are afraid of, and one line to hold onto.",
			"Read Psalm 34 together. Share a time you were sure God was near.",
			"Read Psalm 46 together. Share what “be still” would look like for you this week.",
			"Read Psalm 51 together. Share a line that you needed to hear today.",
			"Read Psalm 63 together. Share what you are hungry and thirsty for.",
			"Read Psalm 91 together. Share where you have felt kept.",
			"Read Psalm 100 together. Share three things you would sing about.",
			"Read Psalm 121 together. Share who you are asking God to watch over.",
			"Read Psalm 139 together. Share what it means to be fully known, and still loved.",
			"Read Psalm 145 together. Share the line you want to carry into next week.",
		},
	},
	{
		Key:      "ten-conversations",
		Title:    "Ten conversations",
		Blurb:    "Ten questions to ask each other, one at a time.",
		Category: CategoryConnection,
		Prompts: []string{
			"What is one moment from this year you would live again?",
			"What did you want to be when you were small, and what part of that is still true?",
			"What is something you have been carrying that you have not said out loud?",
			"When do you feel most close to me?",
			"What does a good, ordinary day look like for you?",
			"What is one thing you would like us to do more of?",
			"How did your family handle disagreements, and what do you want to keep or leave behind?",
			"What are you hoping for that you have not told anyone?",
			"How can I pray for you this month?",
			"What do you want us to remember about these ten days?",
		},
	},
	{
		Key:      "advent",
		Title:    "Advent together",
		Blurb:    "Twenty-four days of waiting and welcoming, one small thing each day.",
		Season:   SeasonAdvent,
		Category: CategorySeason,
		Prompts: []string{
			"Light a candle, or just sit in a dim room. Read Isaiah 9:2 and say what you are waiting for.",
			"Read Isaiah 9:6 together. Which of those names do you need most this year?",
			"Tell each other one thing you are hoping for this season, small or large.",
			"Read Luke 1:26–38. Talk about a time you said yes to something you did not understand.",
			"Do something for someone who will not be able to repay you.",
			"Pray for anyone you know who finds this time of year heavy.",
			"Read Luke 1:39–45 together. Who is someone whose company makes you feel less alone?",
			"Write down one thing you are leaving behind this year, and say it to each other.",
			"Read Luke 1:46–55 together, Mary’s song. What would your own song sound like?",
			"Give thanks together for three things that got you here.",
			"Read Isaiah 40:1–5 together. What needs comfort in your life right now?",
			"Send a message to someone you have not spoken to in a while.",
			"Read Luke 1:67–79 together. Share a way light has come to you this year.",
			"Sit in silence together for five minutes. Then say one word for how you feel.",
			"Read Micah 5:2 together. Talk about something small that turned out to matter.",
			"Choose one tradition you want to keep as a couple, and one you want to begin.",
			"Read Matthew 1:18–25 together. Share a time you had to trust without seeing the whole plan.",
			"Give something away that you have been holding onto.",
			"Read Luke 2:1–7 together. Where has God met you in an ordinary or crowded place?",
			"Pray for your families, by name, one by one.",
			"Read Luke 2:8–14 together. Share the last time you were told not to be afraid.",
			"Tell each other what you have learned about waiting.",
			"Read Luke 2:15–20 together. Share the thing you most want to tell someone.",
			"Read John 1:1–14 together. Say, in your own words, what you are welcoming.",
		},
	},
	{
		Key:      "lent",
		Title:    "Forty days of Lent",
		Blurb:    "Forty gentle days of turning back, together.",
		Season:   SeasonLent,
		Category: CategorySeason,
		Prompts: []string{
			"Begin quietly. Read Joel 2:12–13 together and pray for a soft heart.",
			"Choose one small thing to set aside for these forty days, and tell each other what it is.",
			"Read Psalm 51:1–10 together. Name one thing you want made clean.",
			"Eat something simple together tonight, and give thanks for it.",
			"Pray for each other by name, for what you know is hard right now.",
			"Read Matthew 4:1–4 together. Talk about what you are truly hungry for.",
			"Rest. Do nothing productive together for an hour today.",
			"Read Psalm 25:4–7 together. Ask God to show you one way to walk this week.",
			"Say sorry to each other for one small thing, and receive it kindly.",
			"Take a walk together without phones, and say what you noticed.",
			"Read Isaiah 58:6–9 together. Choose one person to help this week.",
			"Give something up, or give something away, quietly.",
			"Read Psalm 32 together. Share what lightness feels like after being forgiven.",
			"Tell them one thing you have been too proud to ask for.",
			"Read Luke 15:11–24 together. Talk about a time you came home.",
			"Pray for anyone you have found it hard to forgive.",
			"Read Psalm 130 together. Say what you are waiting for, and how it feels to wait.",
			"Cook or share a meal for someone else this week.",
			"Read John 3:14–17 together. Share what it means that you were loved first.",
			"Sit somewhere quiet and read Psalm 42 aloud, taking turns.",
			"Write each other a short note about what you have seen in them this season.",
			"Read Romans 5:1–5 together. Share something hard that has taught you patience.",
			"Pray for your church, or for whoever you gather with.",
			"Read Isaiah 43:1–3 together. Say your name aloud, and what it means to be called by it.",
			"Choose one habit you would like to begin, and keep it for the days that remain.",
			"Read Psalm 121 together. Share who you are asking God to watch over.",
			"Tell each other one thing you are learning about yourself these weeks.",
			"Read Philippians 2:5–8 together. Do one small, humble thing for each other today.",
			"Pray for peace in a place that has none.",
			"Read Psalm 22:1–5 together. Say honestly where you have felt far from God.",
			"Read Luke 22:39–46 together. Sit with each other for a while, without trying to fix anything.",
			"Look through old photographs, and give thanks for what you have come through.",
			"Read Isaiah 53:3–5 together. Be quiet for five minutes afterwards.",
			"Read John 13:1–15 together. Wash something for each other: feet, hands, or dishes.",
			"Read Luke 23:33–43 together. Say who you want to forgive before this season ends.",
			"Spend the evening quietly. Turn off the screens and the music.",
			"Read Psalm 88 together, and let it be as sad as it is. Pray simply.",
			"Read Matthew 27:57–61 together. Wait with each other.",
			"Keep today plain and unhurried. Say one thing you are grateful was not lost.",
			"Read Luke 24:1–12 together. Tell each other what you are hoping is coming.",
		},
	},
}

func Templates() []Template { return templates }

func TemplateByKey(key string) (Template, error) {
	for _, t := range templates {
		if t.Key == key {
			return t, nil
		}
	}
	return Template{}, ErrUnknownTemplate
}

// Custom is a challenge a couple wrote themselves.
type Custom struct {
	Title   string
	Prompts []string
}

// ValidateCustom trims what was written and turns it into a Template filed
// under "custom". A field error names the field the same way a client sent
// it, so it can be put next to the right input.
func ValidateCustom(c Custom) (Template, error) {
	fields := map[string]string{}

	title := strings.TrimSpace(c.Title)
	if title == "" || utf8.RuneCountInString(title) > MaxCustomTitle {
		fields["custom.title"] = "Give it a name, up to 80 characters."
	}

	prompts := make([]string, 0, len(c.Prompts))
	for _, p := range c.Prompts {
		prompts = append(prompts, strings.TrimSpace(p))
	}
	switch {
	case len(prompts) < MinCustomDays || len(prompts) > MaxCustomDays:
		fields["custom.prompts"] = fmt.Sprintf("Add between %d and %d days.", MinCustomDays, MaxCustomDays)
	default:
		for _, p := range prompts {
			if p == "" || utf8.RuneCountInString(p) > MaxCustomPrompt {
				fields["custom.prompts"] = "Each day needs some words, up to 200 characters."
				break
			}
		}
	}
	if len(fields) > 0 {
		return Template{}, apperr.Validation(fields)
	}
	return Template{Key: CustomKey, Title: title, Prompts: prompts}, nil
}

// ValidateMark requires done or skipped; explicitly false means un-marking.
func ValidateMark(done, skipped *bool) (Mark, bool, error) {
	switch {
	case done != nil && *done:
		return MarkDone, true, nil
	case skipped != nil && *skipped:
		return MarkSkipped, true, nil
	case done != nil || skipped != nil:
		// Explicitly false: they are taking it back.
		return "", false, nil
	default:
		return "", false, apperr.Validation(map[string]string{
			"done": "Say whether it is done or skipped.",
		})
	}
}

// Entry is what one person is saying about one day: a mark, taking a mark
// back, a note, or any of those together.
type Entry struct {
	// SetMark with Mark records it; ClearMark takes it back; neither leaves
	// the mark alone (a note on its own).
	Mark      Mark
	SetMark   bool
	ClearMark bool
	// nil leaves the note alone; "" clears it.
	Note *string
}

// ValidateEntry reads a day request. It needs at least one of done, skipped
// or note — a request saying nothing is the client's mistake, not a no-op.
func ValidateEntry(done, skipped *bool, note *string) (Entry, error) {
	var e Entry
	if note != nil {
		trimmed := strings.TrimSpace(*note)
		if utf8.RuneCountInString(trimmed) > MaxNoteRunes {
			return Entry{}, apperr.Validation(map[string]string{"note": "Keep it under 280 characters."})
		}
		e.Note = &trimmed
	}
	if done == nil && skipped == nil && note != nil {
		return e, nil
	}
	mark, marked, err := ValidateMark(done, skipped)
	if err != nil {
		return Entry{}, err
	}
	e.Mark, e.SetMark, e.ClearMark = mark, marked, !marked
	return e, nil
}

// ValidateReflection trims a reflection; empty is allowed and means remove it.
func ValidateReflection(text string) (string, error) {
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) > MaxReflectionRunes {
		return "", apperr.Validation(map[string]string{"text": "Keep it under 1000 characters."})
	}
	return text, nil
}

const msgStartRange = "Pick a day from today to two months ahead."

// ParseDay reads a calendar day the way a client sends it, YYYY-MM-DD, held at
// midnight UTC like every other couple-local date here.
func ParseDay(text string) (time.Time, error) {
	day, err := time.Parse(time.DateOnly, strings.TrimSpace(text))
	if err != nil {
		return time.Time{}, apperr.Validation(map[string]string{"started_on": msgStartRange})
	}
	return day, nil
}

// ValidateStart is whether `on` is today or up to MaxStartAhead days after it,
// both in the couple's own calendar.
func ValidateStart(today, on time.Time) error {
	if on.Before(today) || on.After(today.AddDate(0, 0, MaxStartAhead)) {
		return apperr.Validation(map[string]string{"started_on": msgStartRange})
	}
	return nil
}

// ValidateKind reads a kind a client sent; empty is together.
func ValidateKind(k string) (Kind, error) {
	switch Kind(strings.TrimSpace(k)) {
	case "", KindTogether:
		return KindTogether, nil
	case KindMine:
		return KindMine, nil
	}
	return "", apperr.Validation(map[string]string{"kind": "Choose together or just me."})
}

// Edit is a change to a challenge's own details; a nil field is left alone.
type Edit struct {
	Title     *string
	StartedOn *time.Time
	Kind      *Kind
}

// ValidateEdit trims the title; the rules that depend on the challenge
// itself are Challenge.CheckEdit's.
func ValidateEdit(e Edit) (Edit, error) {
	if e.Title != nil {
		title := strings.TrimSpace(*e.Title)
		if title == "" || utf8.RuneCountInString(title) > MaxCustomTitle {
			return Edit{}, apperr.Validation(map[string]string{"title": "Give it a name, up to 80 characters."})
		}
		e.Title = &title
	}
	return e, nil
}

// CheckEdit decides whether this person may make this change to this
// challenge as it stands, and returns what actually changes: a field sent back
// as it already is is dropped, so a client that always sends everything is
// never refused for it. Refused in order: not theirs to touch, over, then the
// field rules.
func (c Challenge) CheckEdit(me uuid.UUID, e Edit) (Edit, error) {
	if !c.MayTouch(me) {
		return Edit{}, ErrNoSuchChallenge
	}
	if c.Status != StatusActive {
		return Edit{}, ErrOver
	}
	if e.Title != nil && *e.Title == c.Title {
		e.Title = nil
	}
	if e.StartedOn != nil {
		if e.StartedOn.Equal(c.StartedOn) {
			e.StartedOn = nil
		} else {
			if c.Begun() {
				return Edit{}, ErrBegun
			}
			if err := ValidateStart(c.Today, *e.StartedOn); err != nil {
				return Edit{}, err
			}
		}
	}
	if e.Kind != nil {
		switch {
		case *e.Kind == c.Kind:
			e.Kind = nil
		case c.CreatedBy != me:
			return Edit{}, apperr.Validation(map[string]string{"kind": "Only whoever started it can change that."})
		case *e.Kind == KindMine && c.anyoneElseMarked(me):
			return Edit{}, ErrKindLocked
		}
	}
	return e, nil
}

// ValidatePlan trims the day texts a plan is being replaced with.
func ValidatePlan(prompts []string) ([]string, error) {
	out := make([]string, 0, len(prompts))
	for _, p := range prompts {
		out = append(out, strings.TrimSpace(p))
	}
	if len(out) < MinCustomDays || len(out) > MaxCustomDays {
		return nil, apperr.Validation(map[string]string{"prompts": fmt.Sprintf("Add between %d and %d days.", MinCustomDays, MaxCustomDays)})
	}
	for _, p := range out {
		if p == "" || utf8.RuneCountInString(p) > MaxCustomPrompt {
			return nil, apperr.Validation(map[string]string{"prompts": "Each day needs some words, up to 200 characters."})
		}
	}
	return out, nil
}

// CheckPlan decides whether this person may replace the days with `prompts`:
// extending is always fine; shortening drops days from the end, and only
// while none of those has a mark or a note from anyone.
func (c Challenge) CheckPlan(me uuid.UUID, prompts []string) error {
	if !c.MayTouch(me) {
		return ErrNoSuchChallenge
	}
	if c.Status != StatusActive {
		return ErrOver
	}
	for _, d := range c.Days {
		if d.N > len(prompts) && (len(d.Marks) > 0 || len(d.Notes) > 0) {
			return ErrDayHasMarks(d.N)
		}
	}
	return nil
}

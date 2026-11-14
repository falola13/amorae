package questions

import (
	"bytes"
	"crypto/sha256"
	"sort"
	"time"

	"github.com/google/uuid"
)

// Pick is the question a couple is asked today. A question counts as used
// only once one of them has answered it, so a skipped day does not burn one.
// The bank is walked in an order that is the couple's own (a stable shuffle
// seeded from their id, so two couples are not asked the same thing on the
// same morning) and the first unused question wins. When every question has
// been used, the one whose latest use is oldest comes round again.
//
// lastAsked is each used question's most recent day, as stored with the
// answers. Questions that are no longer in the bank are ignored.
func Pick(bank []string, coupleID uuid.UUID, lastAsked map[string]time.Time) string {
	if len(bank) == 0 {
		return ""
	}
	order := coupleOrder(bank, coupleID)
	var oldest string
	var oldestAt time.Time
	for n, q := range order {
		at, used := lastAsked[q]
		if !used {
			return q
		}
		if n == 0 || at.Before(oldestAt) {
			oldest, oldestAt = q, at
		}
	}
	return oldest
}

// coupleOrder is the bank in this couple's own order. It sorts by a hash of
// the couple and the question text, so adding to the bank never reshuffles
// what comes next for questions already in it.
func coupleOrder(bank []string, coupleID uuid.UUID) []string {
	type keyed struct {
		q   string
		key [sha256.Size]byte
	}
	ks := make([]keyed, len(bank))
	for i, q := range bank {
		ks[i] = keyed{q: q, key: sha256.Sum256(append(coupleID[:len(coupleID):len(coupleID)], q...))}
	}
	sort.Slice(ks, func(a, b int) bool {
		if c := bytes.Compare(ks[a].key[:], ks[b].key[:]); c != 0 {
			return c < 0
		}
		return ks[a].q < ks[b].q
	})
	out := make([]string, len(ks))
	for i, k := range ks {
		out[i] = k.q
	}
	return out
}

// bank is the list the questions are drawn from. Its order does not matter
// (each couple walks it in their own order), and the question is stored with
// the answer, so editing it never changes what was already answered.
var bank = []string{
	// Light
	"What is a small thing that made you smile this week?",
	"What would your perfect lazy Saturday look like, start to finish?",
	"Which meal could you eat every week for a year and never get tired of?",
	"What song is stuck in your head lately, and does it deserve to be?",
	"If we had a free afternoon and no plans, what should we do with it?",
	"What is the most ridiculous thing you have ever been scared of?",
	"What is one thing I do that always makes you laugh, even when you are trying not to?",
	"What is your go-to comfort food when the day has been long?",
	"If you could have any skill by tomorrow morning, what would it be?",
	"What was the best part of your day so far, even if it was tiny?",
	"What is a habit of mine that you have secretly grown to love?",
	"Which of our inside jokes is your favourite?",
	"If we were stuck in traffic for three hours, what would you want us to talk about?",
	"What is something you are looking forward to this month?",
	"What is the funniest thing that has happened to you recently?",
	"If you could live anywhere in Nigeria for a year, where would it be?",
	"What is a smell that takes you straight back to childhood?",
	"What is something simple that instantly improves your mood?",
	"What would you cook for me if you wanted to impress me?",
	"What is the best gift you have ever received, and what made it so good?",
	"What is a film or series you could rewatch forever?",
	"What is one small luxury you would never give up?",
	"What did you want to be when you were ten?",
	"What is something you are surprisingly good at?",
	"If tonight were totally ours, what ordinary thing would you want to do together?",
	"What is your favourite way for me to say good morning?",
	"What is a food you hated as a child and love now?",
	"What is the most comfortable you have ever felt in a place?",
	"Which of my clothes do you secretly like the most?",
	"What is one thing you would like us to try together, just once?",
	// Deep
	"When do you feel most understood by me?",
	"What is something you find hard to ask me for?",
	"What is a worry you have been carrying quietly lately?",
	"When have you felt most proud of yourself, and did anyone notice?",
	"What do you need more of from me right now?",
	"What is something you wish I knew about how your mind works?",
	"What helps you most when you are overwhelmed?",
	"What is a fear you have had about us, and how has it changed?",
	"When was the last time you felt really close to me?",
	"What is something from your past that still shapes how you love?",
	"What makes you feel safe with me?",
	"What is one thing you are still learning to forgive yourself for?",
	"How do you like me to show up when you are having a hard day?",
	"What is something about your family that has shaped the way you see marriage?",
	"What is a boundary that matters to you, and have I respected it well?",
	"What do you think we handle really well as a couple?",
	"What is one thing we could do better when we disagree?",
	"What kind of rest do you actually need, and do you let yourself have it?",
	"When do you feel most like yourself?",
	"What is something you are afraid to hope for?",
	"What is a compliment you rarely believe, and what would help you believe it?",
	"What did you need to hear as a child that no one said?",
	"How do you know when you are being loved well?",
	"What is something about money that stresses you, even a little?",
	"What is a way I have changed you for the better?",
	"What part of your work or study weighs on you the most right now?",
	"What do you wish we talked about more?",
	"What is one thing you admire about how I handle hard things?",
	"What is something you have never told me because it seemed too small?",
	"How do you want us to handle disagreements when we are both tired?",
	// Memory
	"What is your very first memory of me?",
	"What did you think of me the first time we met?",
	"What is the best day we have spent together so far?",
	"When did you first realise this was serious?",
	"What is a moment from our early days that you replay?",
	"What is the best meal we have shared?",
	"What is a trip or outing we took that you would happily repeat?",
	"What is something I said to you long ago that you never forgot?",
	"What was the first thing you noticed about me?",
	"What was the hardest stretch we have been through, and what got us out of it?",
	"What is a photo of us that you love, and what is happening in it?",
	"Where were you the day you knew you wanted to marry me?",
	"What is a small kindness I did that you still remember?",
	"What is a memory from your childhood you would love to share with our children?",
	"What is your favourite family gathering you have ever been to?",
	"Who in your life taught you the most about love, and how?",
	"What is something we did early on that we should start doing again?",
	"What is the most thoughtful thing anyone has done for you?",
	"What is a place that holds good memories for you, and why?",
	"What was the best surprise we have had together?",
	// Future
	"Where do you picture us five years from now?",
	"What do you want our home to feel like when people walk in?",
	"What is one dream you have not told many people about?",
	"What would you like us to be doing every year, as a tradition?",
	"What kind of parents, or aunties and uncles, do you hope we will be?",
	"What is something you would like to have learned by this time next year?",
	"What would a good ordinary Tuesday look like for us in ten years?",
	"What is one place you want us to see together?",
	"What do you hope people say about our marriage?",
	"What is a goal we could start working towards this month?",
	"How would you like us to spend our anniversaries?",
	"What is one thing you would like to be braver about?",
	"What do you hope we never stop doing?",
	"What would you want to build together, if money and time were not a problem?",
	"What is something you want us to do before we are old?",
	"How would you like us to handle our finances as a team?",
	"What do you want the first hour of our mornings to look like?",
	"What is a lesson you want to pass on to the people who look up to us?",
	"What does a good friendship outside of us look like, and how can I support yours?",
	"What is one thing you want us to be known for?",
	// Gratitude
	"What is something I did this week that you are thankful for?",
	"What is a quality of mine you are grateful for on difficult days?",
	"What is one thing about our life right now that you would not trade?",
	"Who has been good to us lately, and how could we thank them?",
	"What is something ordinary in your day that you are grateful for?",
	"What is something I do that makes your life easier?",
	"What is a sacrifice I have made that you do not want me to think went unnoticed?",
	"What is a thing about our home that you are happy about?",
	"What is the kindest thing I have said to you?",
	"What is something about your health, work, or family you are thankful for today?",
	"How have I helped you become more of who you want to be?",
	"What part of our routine do you cherish most?",
	"What is something good that came out of a hard season for us?",
	"What did you learn from me this year?",
	"What is one thing you appreciate about my family?",
	// Faith
	"What is a prayer of yours that God has answered, even in a way you did not expect?",
	"What is a verse that has carried you through a difficult time?",
	"How can I pray for you this week?",
	"When did you last feel God very near, and what was happening?",
	"What would you like our prayer life together to look like?",
	"What does a good Sunday look like for you, from morning to evening?",
	"What is something you are trusting God with right now?",
	"What is one thing from a sermon or a conversation lately that stayed with you?",
	"How has your faith changed since we met?",
	"What is a hymn or worship song that undoes you, in a good way?",
	"What would you like us to serve or give towards, together?",
	"What is something you wish you were more patient about, and how can we pray over it?",
	"Which person of faith has shaped you most, and what did they show you?",
	"What do you want our home to be known for, spiritually?",
	"What is one way I can make it easier for you to rest in God?",
	"What does it look like for us to bring our worries to God before we bring them to each other?",
	// A few more of each, for balance
	"What is something you find beautiful that most people walk past?",
	"What is a rule of ours that nobody else would understand?",
	"What would you like to hear me say more often?",
	"What do you remember about the way your parents or guardians showed love?",
	"If we could pause the world for one hour, how would you spend it with me?",
	"What is something you want to be better at as a partner?",
	"What is the first thing you would do with an unexpected free week?",
	"Which of your friends do you hope I get to know better, and why?",
}

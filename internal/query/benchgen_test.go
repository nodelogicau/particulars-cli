package query

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

// genVersion names the generator's shape. Bump it whenever the shape below
// changes, so results from different shapes are never compared as one.
const genVersion = "g1"

// genSeed is the seed every scale benchmark generates from.
const genSeed = 20260912

// genShape holds the proportions of a generated workspace, taken from the
// two dogfood workspaces (2026-10-16): particulars-knowledge has 182 claims
// over 9 particulars with 43 syntheses; particulars-personal has 192 claims
// over 54 particulars with 16 syntheses and one merge. Both retract 1–3% of
// claims. Topic vocabularies ran from a quarter of the claim count to more
// than the claim count.
var genShape = struct {
	claimsPerParticular  int     // particulars = claims / this
	synthesisEvery       [2]int  // a particular is synthesised after this many pending claims (inclusive range)
	retractFraction      float64 // of claims
	particularsPerMerge  int     // merges = particulars / this, at least one when there are two particulars
	topicsPerClaim       [2]int  // inclusive range
	topicVocabPerClaims  int     // vocabulary = claims / this
	aliasFraction        float64 // of particulars
	contentWords         [2]int  // inclusive range
	zipfSubject, zipfTag float64 // Zipf exponents for subject and topic choice
}{
	claimsPerParticular: 10,
	synthesisEvery:      [2]int{3, 9},
	retractFraction:     0.02,
	particularsPerMerge: 50,
	topicsPerClaim:      [2]int{1, 3},
	topicVocabPerClaims: 3,
	aliasFraction:       0.3,
	contentWords:        [2]int{12, 40},
	zipfSubject:         1.2,
	zipfTag:             1.1,
}

var genWords = strings.Fields(`service deploys runs uses stores reads writes owns depends replaces
migrated retired version release cluster region database cache queue index schema token session
timeout limit budget contract team review policy incident outage latency throughput customer
vendor billing invoice audit compliance backup restore failover replica shard partition`)

// genStats summarises what a generation wrote.
type genStats struct {
	Particulars, Claims, Syntheses, Retracted, Merges, Topics int
	// HotParticular has the most claims; HotTopic is carried by the most.
	HotParticular, HotTopic string
	Files                   int
}

// generator mints deterministic ids and timestamps from one seeded source.
type generator struct {
	r   *rand.Rand
	now time.Time
}

func (g *generator) tick() time.Time {
	g.now = g.now.Add(time.Duration(1+g.r.IntN(120)) * time.Minute)
	return g.now
}

// uuidAt builds a canonical UUIDv7 for t with seeded random bits.
func (g *generator) uuidAt(t time.Time) string {
	var u uuid.UUID
	ms := uint64(t.UnixMilli())
	for i := 0; i < 6; i++ {
		u[i] = byte(ms >> (8 * (5 - i)))
	}
	for i := 6; i < 16; i++ {
		u[i] = byte(g.r.UintN(256))
	}
	u[6] = 0x70 | (u[6] & 0x0f)
	u[8] = 0x80 | (u[8] & 0x3f)
	return u.String()
}

func (g *generator) id(t dkf.Type, at time.Time) string {
	return dkf.Prefix(t) + "_" + g.uuidAt(at)
}

func (g *generator) between(r [2]int) int { return r[0] + g.r.IntN(r[1]-r[0]+1) }

func (g *generator) content(subject string, n int) string {
	words := make([]string, g.between(genShape.contentWords))
	for i := range words {
		words[i] = genWords[g.r.IntN(len(genWords))]
	}
	return fmt.Sprintf("%s fact %d: %s.", subject, n, strings.Join(words, " "))
}

// generateWorkspace writes a workspace of the given claim count into dir from
// seed. Every object is written through the store, so files are exactly what
// the CLI would write; the index is rebuilt once at the end.
func generateWorkspace(dir string, claims int, seed uint64) (genStats, error) {
	g := &generator{r: rand.New(rand.NewPCG(seed, uint64(claims))), now: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
	cfg := store.NewConfig()
	cfg.Workspace.ID = g.uuidAt(g.now)
	w, err := store.Init(dir, cfg)
	if err != nil {
		return genStats{}, err
	}
	src := dkf.Source{Harness: "bench-gen"}
	var st genStats

	nPar := max(2, claims/genShape.claimsPerParticular)
	pars := make([]*dkf.Particular, nPar)
	for i := range pars {
		label := fmt.Sprintf("Entity %05d", i)
		p := &dkf.Particular{ID: g.id(dkf.TypeParticular, g.tick()), URI: dkf.MintURI("", cfg.Workspace.ID, dkf.Slugify(label)), Label: label}
		if g.r.Float64() < genShape.aliasFraction {
			p.Aliases = []string{fmt.Sprintf("E%d", i)}
		}
		if err := w.WriteParticular(p); err != nil {
			return st, err
		}
		pars[i] = p
	}
	st.Particulars = nPar

	vocab := max(5, claims/genShape.topicVocabPerClaims)
	subjectZ := rand.NewZipf(g.r, genShape.zipfSubject, 1, uint64(nPar-1))
	topicZ := rand.NewZipf(g.r, genShape.zipfTag, 1, uint64(vocab-1))

	type pending struct {
		claims  []string
		current string
		due     int
	}
	state := make([]pending, nPar)
	for i := range state {
		state[i].due = g.between(genShape.synthesisEvery)
	}
	claimCount := make([]int, nPar)
	topicCount := map[string]int{}

	for n := 0; n < claims; n++ {
		pi := int(subjectZ.Uint64())
		p := pars[pi]
		at := g.tick()
		seen := map[string]bool{}
		var topics []string
		for k := g.between(genShape.topicsPerClaim); k > 0; k-- {
			t := fmt.Sprintf("topic-%04d", topicZ.Uint64())
			if !seen[t] {
				seen[t] = true
				topics = append(topics, t)
				topicCount[t]++
			}
		}
		c := &dkf.Claim{ID: g.id(dkf.TypeClaim, at), Subject: p.ID, Content: g.content(p.Label, n), Source: src,
			Context: dkf.Context{Scope: dkf.ScopePersonal, Topics: topics}, Timestamp: at, Evidential: dkf.EvidentialObserved}
		retract := g.r.Float64() < genShape.retractFraction
		if retract {
			c.Retracted = &dkf.Retracted{Timestamp: at.Add(time.Hour), Reason: "superseded in generation", Source: src, Kind: dkf.KindSupersession}
			st.Retracted++
		}
		if err := w.Create(c); err != nil {
			return st, err
		}
		claimCount[pi]++
		st.Claims++
		if retract {
			continue
		}

		s := &state[pi]
		s.claims = append(s.claims, c.ID)
		if len(s.claims) < s.due {
			continue
		}
		var inputs []dkf.Input
		if s.current != "" {
			inputs = append(inputs, dkf.Input{ID: s.current, Role: dkf.RoleThesis, Weight: dkf.WeightPrimary})
		}
		for i, id := range s.claims {
			role := dkf.RoleAntithesis
			if i == 0 && s.current == "" {
				role = dkf.RoleThesis
			}
			inputs = append(inputs, dkf.Input{ID: id, Role: role})
		}
		sat := g.tick()
		syn := &dkf.Synthesis{ID: g.id(dkf.TypeSynthesis, sat), Subject: p.ID, Content: g.content(p.Label, n), Inputs: inputs,
			Unresolved: "None identified", Source: src, Method: dkf.DefaultMethod, Timestamp: sat,
			Context: dkf.Context{Scope: dkf.ScopePersonal}}
		if err := w.Create(syn); err != nil {
			return st, err
		}
		st.Syntheses++
		s.current, s.claims, s.due = syn.ID, nil, g.between(genShape.synthesisEvery)
	}

	// Disjoint pairs, so no merge class has more than two members.
	nMerge := max(1, nPar/genShape.particularsPerMerge)
	perm := g.r.Perm(nPar)
	for i := 0; i < nMerge && 2*i+1 < len(perm); i++ {
		a, b := pars[perm[2*i]].URI, pars[perm[2*i+1]].URI
		if a > b {
			a, b = b, a
		}
		at := g.tick()
		m := &dkf.Merge{ID: g.id(dkf.TypeMerge, at), URIs: []string{a, b}, Reason: "same entity", Source: src, Timestamp: at}
		if err := w.Create(m); err != nil {
			return st, err
		}
		st.Merges++
	}

	if _, err := w.RebuildIndex(); err != nil {
		return st, err
	}

	st.Topics = len(topicCount)
	hot := 0
	for i, n := range claimCount {
		if n > claimCount[hot] {
			hot = i
		}
	}
	st.HotParticular = pars[hot].ID
	for t, n := range topicCount {
		if st.HotTopic == "" || n > topicCount[st.HotTopic] || (n == topicCount[st.HotTopic] && t < st.HotTopic) {
			st.HotTopic = t
		}
	}
	st.Files = st.Particulars + st.Claims + st.Syntheses + st.Merges + 2 // + dkf.yaml, index.yaml
	return st, nil
}

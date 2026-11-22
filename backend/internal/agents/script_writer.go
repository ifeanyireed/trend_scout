package agents

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"trendscout-backend/internal/ai"
)

const (
	MaxShotDurationSeconds = 7.0
	MinShotDurationSeconds = 3.0
	MaxWordsPerShot        = 17
	WordsPerSecondTarget   = 2.45
)

type ScriptShot struct {
	ShotNumber      int     `json:"shotNumber"`
	Timestamp       string  `json:"timestamp"`
	StartSeconds    float64 `json:"startSeconds"`
	EndSeconds      float64 `json:"endSeconds"`
	DurationSeconds float64 `json:"durationSeconds"`
	Visual          string  `json:"visual"`
	Narration       string  `json:"narration"`
	AudioCue        string  `json:"audioCue,omitempty"`
	LocationRef     string  `json:"locationRef,omitempty"`
	ChapterNumber   int     `json:"chapterNumber"`
	ChapterTitle    string  `json:"chapterTitle"`
}

type ScriptSection struct {
	ActNumber       int          `json:"actNumber"`
	ChapterNumber   int          `json:"chapterNumber"`
	Title           string       `json:"title"`
	TimeRange       string       `json:"timeRange"`
	DomainField     string       `json:"domainField"`
	CentralQuestion string       `json:"centralQuestion,omitempty"`
	Paragraphs      []string     `json:"paragraphs"`
	Shots           []ScriptShot `json:"shots"`
	ClaimID         string       `json:"claimId"`
	SourceID        string       `json:"sourceId"`
	CitationURL     string       `json:"citationUrl"`
}

type CharacterEraTemplate struct {
	Role       string
	AgeRange   string
	Appearance string
	Hair       string
	Clothing   string
	Body       string
	Locations  string
	Palette    string
}

type BiographicalDomainProfile struct {
	Key                 string
	Label               string
	OpeningRoom         string
	OriginRoom          string
	ReckoningRoom       string
	CentralQuestion     string
	ClosingQuestion     string
	ChapterTitles       [7]string
	AmbientCues         []string
	ChapterPrimaryShots [7][]string
	ChapterCutawayShots [7][]string
	ChapterLocations    [7]string
	CharacterEras       [3]CharacterEraTemplate
}

type BiographyInput struct {
	PersonName        string
	Category          string
	Country           string
	ProjectTitle      string
	Summary           string
	Biography         string
	Headline          string
	EventTitle        string
	Origins           string
	EarlyAmbitions    string
	Disadvantages     string
	Rejections        string
	Obstacles         string
	TurningPoints     string
	CareerProgression string
	Breakthrough      string
	Claims            []ScriptClaimRef
	TimelineEntries   []string
	CompetitiveBrief  *CompetitivePositioningBrief
}

type ScriptClaimRef struct {
	ClaimID  string
	SourceID string
	Category string
	Text     string
	URL      string
	Status   string
}

// ResolveBiographicalDomain dynamically classifies the subject's biographical field from an explicit
// override, category metadata, and biographical dossier text.
func ResolveBiographicalDomain(category, biography, summary, headline, fieldOverride string) BiographicalDomainProfile {
	override := strings.ToLower(strings.TrimSpace(fieldOverride))
	if override != "" && override != "auto" && override != "auto-detect" {
		switch {
		case strings.Contains(override, "tech") || strings.Contains(override, "software") || strings.Contains(override, "ai"):
			return domainTechInnovation()
		case strings.Contains(override, "business") || strings.Contains(override, "founder") || strings.Contains(override, "finance"):
			return domainBusinessFounders()
		case strings.Contains(override, "science") || strings.Contains(override, "med") || strings.Contains(override, "research"):
			return domainScienceMedicine()
		case strings.Contains(override, "politic") || strings.Contains(override, "law") || strings.Contains(override, "govern"):
			return domainPoliticsLaw()
		case strings.Contains(override, "sport") || strings.Contains(override, "athlet") || strings.Contains(override, "chess"):
			return domainSportsCompetition()
		case strings.Contains(override, "art") || strings.Contains(override, "film") || strings.Contains(override, "music") || strings.Contains(override, "culture"):
			return domainArtsCulture()
		case strings.Contains(override, "humanitarian") || strings.Contains(override, "civic") || strings.Contains(override, "activis"):
			return domainHumanitarianCivic()
		}
	}

	corpus := strings.ToLower(strings.Join([]string{category, category, headline, summary, biography}, " "))
	scores := map[string]int{
		"tech":         countKeywordHits(corpus, []string{"software", "algorithm", "startup", "silicon valley", "code", "programming", "social media", "artificial intelligence", "platform", "internet", "engineer", "tech", "app", "website", "computer science", "cyber", "semiconductor", "robotics", "data center", "meta", "ceo", "hackathon"}),
		"business":     countKeywordHits(corpus, []string{"billionaire", "investor", "company", "corporation", "ipo", "wall street", "nasdaq", "shareholder", "venture", "merger", "acquisition", "chief executive", "founder", "enterprise", "banking", "hedge fund", "market cap", "industrialist", "retail", "conglomerate"}),
		"science":      countKeywordHits(corpus, []string{"scientist", "nobel", "laboratory", "researcher", "physics", "biology", "medicine", "vaccine", "clinical", "genome", "professor", "university", "astronomy", "chemistry", "physician", "surgeon", "journal", "peer-reviewed", "discovery", "patent", "neuroscience", "climate"}),
		"politics":     countKeywordHits(corpus, []string{"senator", "president", "prime minister", "parliament", "congress", "governor", "mayor", "court", "judge", "attorney", "diplomat", "minister", "election", "legislation", "constitutional", "supreme court", "prosecutor", "policy", "cabinet", "treaty", "political"}),
		"sports":       countKeywordHits(corpus, []string{"championship", "olympic", "medal", "tournament", "athlete", "grandmaster", "chess", "tennis", "football", "basketball", "boxing", "sprint", "marathon", "stadium", "coach", "world record", "unseeded", "finals", "league", "world cup", "swimming", "gymnastics"}),
		"arts":         countKeywordHits(corpus, []string{"film", "director", "actor", "actress", "musician", "composer", "singer", "album", "novelist", "author", "painter", "architect", "cinema", "theater", "broadway", "oscar", "grammy", "pulitzer", "gallery", "exhibition", "screenplay", "concert", "literature"}),
		"humanitarian": countKeywordHits(corpus, []string{"humanitarian", "activist", "refugee", "civil rights", "ngo", "nonprofit", "philanthrop", "community", "grassroots", "educator", "advocate", "relief", "peace prize", "whistleblower", "journalist", "investigative reporter", "human rights", "charity", "civic"}),
	}

	bestKey := "general"
	bestScore := 0
	// Deterministic priority order in case of ties
	order := []string{"tech", "science", "politics", "business", "arts", "humanitarian", "sports"}
	for _, k := range order {
		if scores[k] > bestScore {
			bestScore = scores[k]
			bestKey = k
		}
	}

	switch bestKey {
	case "tech":
		return domainTechInnovation()
	case "business":
		return domainBusinessFounders()
	case "science":
		return domainScienceMedicine()
	case "politics":
		return domainPoliticsLaw()
	case "sports":
		return domainSportsCompetition()
	case "arts":
		return domainArtsCulture()
	case "humanitarian":
		return domainHumanitarianCivic()
	default:
		return domainGeneralBiography()
	}
}

func countKeywordHits(text string, keywords []string) int {
	total := 0
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			total++
		}
	}
	return total
}

func domainTechInnovation() BiographicalDomainProfile {
	return BiographicalDomainProfile{
		Key:             "tech_innovation",
		Label:           "Technology & Innovation",
		OpeningRoom:     "a brightly lit congressional hearing room lined with microphones and cameras",
		OriginRoom:      "a quiet student desk lit by a single glowing monitor",
		ReckoningRoom:   "a federal courtroom and a legislative committee chamber",
		CentralQuestion: "What happens when a small technical project becomes infrastructure for millions of people—and who answers for it?",
		ClosingQuestion: "What does the builder of global digital infrastructure owe the people who live inside it?",
		ChapterTitles: [7]string{
			"Cold Open",
			"The Prodigy & Early Craft",
			"The First Build",
			"Contested Ground",
			"Scale",
			"Reckoning",
			"The Open Question",
		},
		AmbientCues: []string{
			"Murmur of the hearing room continues",
			"Silence",
			"Low hum of cooling fans inside a server room",
			"Quiet background room ambient",
			"Ambient fadeout",
		},
		ChapterLocations: [7]string{
			"Public hearing chamber & early workroom",
			"Suburban family home & university campus",
			"Late-night dorm or studio workspace & server closet",
			"Federal courthouse & editorial archive desk",
			"Silicon Valley office, stock exchange & global data center",
			"Legislative committee chamber, district court & legal archive",
			"Empty origin room & quiet public hearing chamber",
		},
		ChapterPrimaryShots: [7][]string{
			{
				"Wide shot of a public hearing room with rows of microphones and an expectant press gallery.",
				"Medium shot of the witness table under bright overhead chamber lights.",
				"Over-the-shoulder shot looking past the witness seat toward rows of observers in the gallery.",
				"Wide shot of a quiet brick university residence exterior at night under streetlights.",
				"Interior shot of a dark student room with a glowing computer monitor displaying early code.",
				"Medium shot looking down a quiet corridor toward closed heavy wooden doors.",
			},
			{
				"Wide shot of a peaceful tree-lined residential street in soft morning light.",
				"Exterior shot of a modest family home with sunlit windows and a quiet sidewalk.",
				"CRT or early desktop screen displaying custom software interfaces and raw code windows.",
				"Wide interior shot of an evening university lecture hall with chalkboards covered in algorithms.",
				"Exterior of weathered brick academic buildings under an overcast sky.",
				"Wide shot of a university courtyard in autumn with fallen leaves along stone paths.",
			},
			{
				"Wide shot of a cluttered study desk with stacked manuals, cables, and a single desk lamp.",
				"Low-angle shot of a glowing computer monitor illuminating a silhouette in a dark room.",
				"Screen display showing an early prototype web interface with austere typography.",
				"Wide interior shot of an empty university computer lab with rows of dark desktop monitors.",
				"Desktop screen displaying a rapidly updating registration log of early network users.",
				"Interior of a cramped student room with laptops, notebooks, and printouts across desks.",
			},
			{
				"Formal legal complaint document inside a manila folder resting on a dark oak desk.",
				"Wide exterior shot of a federal courthouse facade with granite columns under grey skies.",
				"Front-page view of an investigative newspaper report spread flat across a wooden table.",
				"Printed court settlement agreement lying open with heavy black redacted lines.",
				"Interior shot of an empty mahogany courtroom with tall ceilings and silent benches.",
				"View looking across empty spectator benches toward an unlit witness stand.",
			},
			{
				"Aerial shot of a quiet technology corridor neighborhood with low-slung offices and trees.",
				"Interior of a converted house workspace filled with folding tables, monitors, and tangled cables.",
				"Late-night interior of an engineering office with glass whiteboards and coffee cups.",
				"Exterior view of a financial exchange building glowing with outdoor market tickers.",
				"Wide shot of a packed urban plaza at night lit by thousands of smartphone screens.",
				"Interior aisle of a modern data center filled with long rows of blue-lit server racks.",
			},
			{
				"Wide exterior shot of a national capitol dome beneath an overcast sky.",
				"Committee hearing room with rows of wooden tables, nameplates, and witness microphones.",
				"Official subpoena and regulatory filing documents resting on a polished conference table.",
				"Smartphone screen held in warm evening light showing an endlessly scrolling social feed.",
				"Television monitor mounted inside an empty courtroom displaying a recorded deposition.",
				"Modern glass-walled corporate conference room with an empty head chair.",
			},
			{
				"Wide shot of an empty origin workspace bathed in quiet morning window light.",
				"Wide shot of an empty legislative hearing room with mahogany tables and silent microphones.",
				"Financial and regulatory summary reports printed in black ink resting side by side on a desk.",
				"Wide shot of an empty marble courthouse hallway leading to closed chamber doors.",
				"Over-the-shoulder view looking past an empty witness table toward vacant gallery seats.",
				"Final wide shot of a dark, quiet room with a single glowing window in the distance.",
			},
		},
		ChapterCutawayShots: [7][]string{
			{
				"Close-up of press camera lenses and silent shutters aimed toward the witness table.",
				"Macro shot of a steel gooseneck microphone capturing movement at the table.",
				"Close-up of hands tightly folded in the gallery audience.",
				"High-angle wide shot looking down over the entire hearing chamber from the balcony.",
				"Low-angle shot looking up at the brick facade of the early residence.",
				"Macro shot of fingers typing steadily on an early mechanical keyboard.",
			},
			{
				"Medium shot of a quiet suburban driveway in early morning shadow.",
				"Close-up of hands plugging an Ethernet or telephone cable into a wall jack.",
				"Macro detail of glowing indicator lights on an early network modem.",
				"Close-up of a mouse cursor clicking through an austere operating system window.",
				"Close-up of white chalk equations written across a dark green chalkboard.",
				"Macro detail of a printed technology magazine review resting under lamp light.",
			},
			{
				"Close-up of a ceramic mug beside handwritten algorithms on a yellow legal pad.",
				"Close-up of a computer monitor displaying raw HTML and backend source lines.",
				"Macro detail of an index finger clicking a beige computer mouse button.",
				"Macro shot of blue Ethernet cables plugged into a flashing network switch.",
				"Close-up of a paper desk calendar with a key launch date circled in ink.",
				"Macro detail of newsprint headlines folded neatly on a wooden surface.",
			},
			{
				"Close-up of an official court stamp embossed on a legal filing.",
				"Macro shot of docket numbers printed across the top margin of a complaint.",
				"Close-up of a pen resting beside a marked newspaper clipping.",
				"Macro detail of heavy black ink redactions across a settlement page.",
				"Close-up of a wooden gavel resting motionless on a judge's bench.",
				"High-angle shot of an empty wooden counsel table inside a quiet courtroom.",
			},
			{
				"Close-up of a brass office key turning in a front door lock.",
				"Macro shot of stacked notebooks and charging cables beside an engineering keyboard.",
				"Close-up of a fountain pen resting beside a corporate term sheet signature line.",
				"Close-up of marker diagrams and network graphs drawn across a dry-erase whiteboard.",
				"Medium shot of street-level financial tickers glowing on outdoor LED screens.",
				"Macro shot of server cooling fans spinning steadily inside a dark equipment rack.",
			},
			{
				"Close-up of engraved brass nameplates arranged along a committee dais.",
				"Close-up of an official government seal printed atop a congressional letter.",
				"Close-up of a thumb scrolling steadily down a glossy smartphone screen.",
				"Macro shot of legal complaint margins highlighted in yellow ink.",
				"Close-up of signature pages at the back of a thick multistate settlement binder.",
				"Medium shot of stacked archival cardboard case boxes inside a quiet legal vault.",
			},
			{
				"Close-up of an empty wooden desk surface illuminated by slanted morning light.",
				"Medium shot of empty witness table chairs facing the congressional panel dais.",
				"Close-up of financial ledger figures printed cleanly in black ink on white paper.",
				"Macro view of a smartphone screen reflecting soft evening room light.",
				"Medium shot of tall double wooden doors leading into an empty court chamber.",
				"Medium close-up of an unlit television monitor resting on a wooden courtroom stand.",
			},
		},
		CharacterEras: [3]CharacterEraTemplate{
			{
				Role:       "Main Subject — Formative Origins",
				AgeRange:   "Youth & Early Technical Apprenticeship",
				Appearance: "Observant, focused demeanor in modest academic and home study environments before public prominence.",
				Hair:       "Natural, unstyled early-era hair matching archival student photographs.",
				Clothing:   "Plain dark sweatshirt, simple cotton tee, worn sneakers, student backpack.",
				Body:       "Relaxed, intense posture hunched slightly toward a glowing terminal or notebook.",
				Locations:  "Suburban home desk, university dorm room, late-night campus computer lab.",
				Palette:    "Cool CRT blue, tungsten desk amber, muted brick red, deep shadow",
			},
			{
				Role:       "Main Subject — Rapid Scale & Institutional Ascent",
				AgeRange:   "Founder & Executive Expansion Era",
				Appearance: "Sharper, composed public presence navigating investor meetings, product launches, and media scrutiny.",
				Hair:       "Clean, understated executive grooming consistent across press briefings.",
				Clothing:   "Minimalist dark crewneck or tailored charcoal blazer over crisp shirt, subtle security badge.",
				Body:       "Measured, controlled posture behind glass conference tables and keynote lecterns.",
				Locations:  "Converted startup office, open-plan engineering floor, exchange balcony, data center corridor.",
				Palette:    "Slate steel, glass reflection white, muted navy, cool architectural grey",
			},
			{
				Role:       "Main Subject — Public Reckoning & Legacy",
				AgeRange:   "Public Testimony & Institutional Accountability Era",
				Appearance: "Weathered composure under harsh committee lighting, subtle tension in eyes and jaw.",
				Hair:       "Neatly trimmed formal grooming under bright press lights.",
				Clothing:   "Dark tailored suit, conservative tie, crisp white collar behind brass witness microphones.",
				Body:       "Upright, guarded posture facing formal committee dais and press cameras.",
				Locations:  "Congressional hearing chamber, federal courthouse corridor, glass executive boardroom.",
				Palette:    "Mahogany wood, brass highlight, overcast marble grey, stark documentary contrast",
			},
		},
	}
}

func domainBusinessFounders() BiographicalDomainProfile {
	p := domainTechInnovation()
	p.Key = "business_founders"
	p.Label = "Business & Founders"
	p.OpeningRoom = "a crowded shareholder hall and regulatory hearing room under harsh lighting"
	p.OriginRoom = "a cramped back-room office stacked with paper ledgers and invoices"
	p.ReckoningRoom = "a federal courthouse and an institutional boardroom"
	p.CentralQuestion = "What happens when a private enterprise grows larger than the rules built to contain it—and who answers for the ledger?"
	p.ClosingQuestion = "When ambition reshapes entire markets, where does innovation end and public accountability begin?"
	p.ChapterTitles = [7]string{
		"Cold Open",
		"Early Apprenticeships",
		"The First Venture",
		"Disputes & Rivals",
		"Empire & Scale",
		"Reckoning",
		"The Open Question",
	}
	p.ChapterLocations = [7]string{
		"Regulatory hearing chamber & early workshop",
		"Family storefront, hometown street & early office",
		"First leased workspace, warehouse & regional bank",
		"Federal courthouse, boardroom & financial press desk",
		"Corporate headquarters, trading floor & international summit",
		"Congressional committee, regulatory audit room & court archive",
		"Empty founding office & quiet public chamber",
	}
	return p
}

func domainScienceMedicine() BiographicalDomainProfile {
	return BiographicalDomainProfile{
		Key:             "science_medicine",
		Label:           "Science & Medicine",
		OpeningRoom:     "a quiet international scientific symposium hall with tiered wooden seats",
		OriginRoom:      "a narrow university basement laboratory illuminated by fluorescent tubes",
		ReckoningRoom:   "a peer-review inquiry board and a public health hearing chamber",
		CentralQuestion: "What happens when one researcher's discovery alters millions of lives—and who governs the consequences?",
		ClosingQuestion: "When scientific breakthroughs outrun the institutions meant to verify them, what endures in the record?",
		ChapterTitles: [7]string{
			"Cold Open",
			"The Early Mind",
			"The Breakthrough Experiment",
			"Skepticism & Priority",
			"Global Impact",
			"Scrutiny & Reckoning",
			"The Open Question",
		},
		AmbientCues: []string{
			"Low hum of laboratory ventilation continues",
			"Silence",
			"Distant murmur of a lecture auditorium",
			"Quiet room ambient",
			"Ambient fadeout",
		},
		ChapterLocations: [7]string{
			"International symposium hall & quiet research lab",
			"Family study, public library & university science hall",
			"Late-night wet laboratory, chalkboard room & instrument bench",
			"Academic review boardroom, journal editorial office & archive",
			"Hospital wing, research institute & global health assembly",
			"Regulatory panel chamber, ethics committee room & clinical archive",
			"Empty laboratory bench & quiet lecture hall",
		},
		ChapterPrimaryShots: [7][]string{
			{
				"Wide shot of a tiered scientific auditorium filled with researchers and press microphones.",
				"Medium shot of a wooden lecture podium standing under focused stage lighting.",
				"Over-the-shoulder shot looking past the podium toward silent rows of scientists and physicians.",
				"Wide exterior shot of a weathered stone university science building at dusk.",
				"Interior shot of a quiet laboratory bench illuminated by a single task lamp.",
				"Medium shot looking down an empty tiled hospital research corridor.",
			},
			{
				"Wide shot of a quiet neighborhood library exterior in soft morning light.",
				"Interior of a family study lined with worn reference books and handwritten notebooks.",
				"Wide interior shot of a university lecture hall with chalkboards dense with equations.",
				"Medium shot of glass beakers, specimen slides, and an older brass microscope on a bench.",
				"Exterior shot of brick university quads under an overcast autumn sky.",
				"Wide shot of a quiet graduate reading room with green-shaded desk lamps.",
			},
			{
				"Wide shot of a cluttered laboratory workbench with pipettes, logbooks, and sample trays.",
				"Low-angle shot of an illuminated laboratory hood glowing inside a dark room.",
				"Medium shot of printed experimental data charts spread across a steel bench.",
				"Wide interior shot of a late-night research wing with only one illuminated window.",
				"Desktop screen displaying raw experimental readouts and sequence tables.",
				"Interior of a narrow university office stacked with manuscript drafts and grant applications.",
			},
			{
				"Printed academic peer-review critique resting inside a manila folder on a wooden desk.",
				"Wide exterior shot of a national academy building with carved stone columns.",
				"Open medical journal pages showing competing citations and footnotes.",
				"Printed patent office filing bearing official stamps and technical diagrams.",
				"Interior shot of an empty university committee room with long oak tables.",
				"View looking across empty faculty chairs toward an unlit projection screen.",
			},
			{
				"Aerial shot of a sprawling modern medical research campus surrounded by trees.",
				"Interior of a bright clinical facility with sterile glass walls and monitoring equipment.",
				"Wide shot of an international health assembly chamber with rows of delegation desks.",
				"Exterior shot of a public hospital entrance under early evening streetlamps.",
				"Wide shot of a cold-storage archive room lined with steel laboratory freezers.",
				"Interior of a modern genomic or diagnostic center humming with automated analyzers.",
			},
			{
				"Wide exterior shot of a government regulatory headquarters beneath an overcast sky.",
				"Public inquiry room with rows of committee desks and silver witness microphones.",
				"Thick clinical audit report resting beside stamped regulatory correspondence.",
				"Interior of a quiet hospital corridor with muted overhead lighting.",
				"Television monitor inside a hearing chamber showing archived technical testimony.",
				"Empty ethics board conference room with glass walls and closed binders.",
			},
			{
				"Wide shot of an empty basement laboratory bench bathed in early morning window light.",
				"Wide shot of an empty scientific auditorium with rows of folded wooden seats.",
				"Bound copies of published papers and inquiry reports resting side by side on a desk.",
				"Wide shot of an empty stone university hallway leading to closed archive doors.",
				"Over-the-shoulder view looking past an empty podium toward silent gallery benches.",
				"Final wide shot of a quiet research room with a single illuminated window at dusk.",
			},
		},
		ChapterCutawayShots: [7][]string{
			{
				"Close-up of press microphones arranged across a polished wooden lectern.",
				"Macro shot of glass slide edges catching cool overhead laboratory light.",
				"Close-up of handwritten margin notes inside a bound laboratory logbook.",
				"High-angle wide shot looking down over the entire lecture hall from the balcony.",
				"Low-angle shot looking up at the stone archway of the science building.",
				"Macro shot of gloved fingers adjusting the focus dial on a laboratory microscope.",
			},
			{
				"Close-up of a fountain pen writing formulas across graph paper.",
				"Macro detail of weathered textbook pages marked with pencil underlines.",
				"Close-up of white chalk dust resting along the wooden tray of a green chalkboard.",
				"Macro shot of glass test tubes aligned in a steel rack.",
				"Close-up of a library catalog card stamped with an early archive date.",
				"Medium shot of rain-streaked university laboratory windows at twilight.",
			},
			{
				"Close-up of a timer ticking beside a row of glass sample vials.",
				"Macro shot of a microscope lens reflecting blue instrument light.",
				"Close-up of red ink corrections marked across a printed manuscript draft.",
				"Macro detail of a stamped grant rejection letter resting on a desk.",
				"Close-up of a desk calendar turned to the month of the breakthrough experiment.",
				"Macro shot of printed chromatogram or data spikes on thermal paper.",
			},
			{
				"Close-up of an official journal editorial seal stamped on correspondence.",
				"Macro shot of highlighted paragraphs inside a disputed scientific publication.",
				"Close-up of spectacles resting atop a thick bound patent specification.",
				"Macro detail of crossed-out hypothesis lines in a spiral lab notebook.",
				"Close-up of a wooden committee gavel resting beside stacked dossiers.",
				"High-angle shot of an empty review table inside a dimly lit conference room.",
			},
			{
				"Close-up of a ceremonial medal case resting closed on dark mahogany.",
				"Macro shot of clinical trial consent forms neatly stacked on a clipboard.",
				"Close-up of vial labels moving steadily along a clean production line.",
				"Medium shot of illuminated laboratory windows across a research complex at night.",
				"Close-up of printed statistical charts showing declining mortality or rising adoption.",
				"Macro shot of cooling fans spinning inside a high-throughput computing rack.",
			},
			{
				"Close-up of engraved nameplates along a public oversight committee dais.",
				"Macro shot of yellow-highlighted lines in a regulatory compliance review.",
				"Close-up of an unsealed legal or institutional inquiry envelope.",
				"Medium shot of empty wooden seating inside a silent public hearing room.",
				"Close-up of hands turning the pages of a thick clinical evidence binder.",
				"Medium shot of archived cardboard record boxes stacked inside a vaulted room.",
			},
			{
				"Close-up of an empty stainless-steel laboratory bench lit by morning sun.",
				"Macro detail of handwritten notes still open inside an archived lab notebook.",
				"Close-up of two contrasting dossiers resting side by side on dark oak.",
				"Medium shot of tall double doors leading into a quiet academic assembly hall.",
				"Close-up of an unlit overhead projector lens inside an empty lecture room.",
				"Medium close-up of an empty wooden chair facing a bright laboratory window.",
			},
		},
		CharacterEras: [3]CharacterEraTemplate{
			{
				Role:       "Main Subject — Formative Student & Apprentice",
				AgeRange:   "Early Academic & Laboratory Years",
				Appearance: "Quiet, absorbed focus inside modest university classrooms and late-night research benches.",
				Hair:       "Simple, natural period-accurate hair.",
				Clothing:   "Plain collared shirt, knit sweater, simple cotton lab coat with pen in breast pocket.",
				Body:       "Poised, leaning intently over notebooks, chalkboards, and laboratory instruments.",
				Locations:  "University basement lab, public reading room, evening lecture hall.",
				Palette:    "Fluorescent laboratory green, chalk slate, warm library amber",
			},
			{
				Role:       "Main Subject — Principal Investigator",
				AgeRange:   "Breakthrough & Institutional Leadership Era",
				Appearance: "Measured authority carrying the fatigue of prolonged experiments and public scrutiny.",
				Hair:       "Neatly kept academic grooming.",
				Clothing:   "Crisp button-down shirt, tweed or dark blazer, institutional ID lanyard.",
				Body:       "Composed, steady presence at laboratory benches and conference podiums.",
				Locations:  "Modern research institute, hospital corridor, international scientific conference.",
				Palette:    "Clinical white, cool steel blue, muted slate grey",
			},
			{
				Role:       "Main Subject — Legacy & Public Record",
				AgeRange:   "Senior Statesperson of Science",
				Appearance: "Calm, reflective expression marked by years of scrutiny and historical consequence.",
				Hair:       "Silver-flecked dignified grooming under soft architectural lighting.",
				Clothing:   "Dark tailored jacket, understated shirt, reading glasses.",
				Body:       "Still, measured posture facing oversight panels and archival cameras.",
				Locations:  "National academy hall, quiet archival library, committee hearing room.",
				Palette:    "Warm mahogany, muted silver, deep archival navy",
			},
		},
	}
}

func domainPoliticsLaw() BiographicalDomainProfile {
	return BiographicalDomainProfile{
		Key:             "politics_law",
		Label:           "Politics, Law & Governance",
		OpeningRoom:     "a hushed legislative chamber and federal courtroom filled with reporters",
		OriginRoom:      "a modest local municipal office stacked with paper case files",
		ReckoningRoom:   "a parliamentary inquiry hall and a supreme court chamber",
		CentralQuestion: "What happens when an outsider steps into the rooms where public power is written—and what remains on the record?",
		ClosingQuestion: "When political ambition collides with institutional accountability, who writes the final verdict?",
		ChapterTitles: [7]string{
			"Cold Open",
			"Early Grounding",
			"The First Campaign",
			"Rivals & The Record",
			"Power & Reach",
			"Reckoning",
			"The Open Question",
		},
		AmbientCues: []string{
			"Quiet murmur of the legislative gallery continues",
			"Silence",
			"Distant footfalls echoing across marble corridors",
			"Quiet background chamber ambient",
			"Ambient fadeout",
		},
		ChapterLocations: [7]string{
			"Legislative hearing chamber & local district office",
			"Hometown courthouse, family dining room & law library",
			"Grassroots campaign room, town hall & local press desk",
			"Federal courthouse, editorial newsroom & statutory archive",
			"National capitol, diplomatic hall & televised briefing room",
			"Oversight inquiry chamber, appellate court & public archive",
			"Empty district office & quiet national chamber",
		},
		ChapterPrimaryShots: [7][]string{
			{
				"Wide shot of a national legislative chamber with tiered wooden desks and press galleries.",
				"Medium shot of a witness lectern draped with microphones under bright chamber lights.",
				"Over-the-shoulder shot looking past the lectern toward rows of lawmakers and spectators.",
				"Wide exterior shot of a modest brick municipal building at dusk.",
				"Interior shot of a small district office desk stacked with legal pads and local newspapers.",
				"Medium shot looking down a quiet marble capitol hallway toward tall wooden doors.",
			},
			{
				"Wide shot of a quiet hometown main street under soft morning light.",
				"Exterior of a modest family house with a shaded porch and quiet yard.",
				"Interior of a university law library lined with leather-bound statute volumes.",
				"Wide interior shot of a local town hall with rows of folding wooden chairs.",
				"Exterior shot of a regional county courthouse beneath an overcast sky.",
				"Wide shot of a quiet campus quadrangle in late afternoon shadow.",
			},
			{
				"Wide shot of a cramped storefront campaign office with wall maps and folding tables.",
				"Medium shot of printed flyers, voter rolls, and rotary or desk phones across a table.",
				"Interior of a community hall where a lone microphone stands before empty seats.",
				"Wide exterior shot of a municipal polling station under early morning rain.",
				"Front page of a regional newspaper resting on a wooden newsroom desk.",
				"Interior of a late-night office illuminated by a single desk lamp over policy memos.",
			},
			{
				"Formal legal filing inside a manila folder resting on a dark mahogany table.",
				"Wide exterior shot of a federal courthouse entrance with tall stone steps.",
				"Front-page view of an investigative newspaper report spread flat under lamp light.",
				"Printed legislative committee transcript lying open with marked passages.",
				"Interior shot of an empty mahogany courtroom with tall ceilings.",
				"View looking across empty gallery benches toward a quiet witness stand.",
			},
			{
				"Aerial shot of a national capital avenue lined with classical stone ministries.",
				"Interior of a formal cabinet conference room with leather chairs and leather blotters.",
				"Wide shot of a press briefing room filled with rows of camera tripods.",
				"Exterior shot of an international summit hall lined with diplomatic flags.",
				"Wide shot of a legislative floor voting board glowing inside a silent chamber.",
				"Interior of a grand marble rotunda illuminated by shafts of afternoon daylight.",
			},
			{
				"Wide exterior shot of a national capitol dome beneath heavy grey clouds.",
				"Oversight committee room with rows of wooden tables and silver microphones.",
				"Official subpoena and ethics inquiry documents lying flat on a dark conference table.",
				"Television monitor mounted inside an empty press room showing archived testimony.",
				"Exterior of an appellate courthouse facade at twilight with lit windows.",
				"Stack of numbered archival case folders lined across steel shelving.",
			},
			{
				"Wide shot of an empty local office desk bathed in clear morning window light.",
				"Wide shot of an empty legislative chamber with dark mahogany benches.",
				"Bound copies of signed legislation and inquiry reports resting side by side.",
				"Wide shot of an empty marble corridor leading to closed chamber doors.",
				"Over-the-shoulder view looking past an empty witness table toward silent seats.",
				"Final wide shot of a quiet dark chamber with a single illuminated tall window.",
			},
		},
		ChapterCutawayShots: [7][]string{
			{
				"Close-up of press camera lenses and silent shutters aimed at the dais.",
				"Macro shot of a brass committee microphone catching subtle movement.",
				"Close-up of clasped hands resting on a wooden gallery rail.",
				"High-angle wide shot looking down over the entire chamber from the press balcony.",
				"Low-angle shot looking up at carved stone courthouse pillars.",
				"Macro shot of fingers turning the corner of a typed briefing page.",
			},
			{
				"Close-up of a brass plaque outside a neighborhood legal office.",
				"Macro detail of underlined lines inside a worn constitutional law textbook.",
				"Close-up of a wooden pen resting atop handwritten town-hall notes.",
				"Macro shot of archival black-and-white civic photographs on a desk.",
				"Close-up of rain droplets running down a county courthouse windowpane.",
				"Medium shot of an empty wooden bench outside a municipal clerk's office.",
			},
			{
				"Close-up of a red marker circling districts on a folded paper map.",
				"Macro detail of stamped ballot tally sheets lying on a counting table.",
				"Close-up of a desk calendar showing the exact election night date.",
				"Macro shot of newsprint ink reading early election returns.",
				"Close-up of a ringing office telephone beside stacked policy binders.",
				"Medium shot of empty folding chairs after a late-night community meeting.",
			},
			{
				"Close-up of an official red court seal embossed on a legal envelope.",
				"Macro shot of docket numbers stamped across the margin of a public filing.",
				"Close-up of a fountain pen resting beside a critical editorial clipping.",
				"Macro detail of heavy black ink redactions across an investigative memo.",
				"Close-up of a wooden gavel resting motionless on a committee dais.",
				"High-angle shot of an empty wooden counsel table inside a quiet courtroom.",
			},
			{
				"Close-up of a fountain pen signing a formal bill on heavy cream paper.",
				"Macro shot of embossed gold lettering on a leather executive briefing folder.",
				"Close-up of television tally lights glowing red inside a briefing room.",
				"Medium shot of diplomatic motorcade lights reflecting off wet avenue pavement.",
				"Close-up of printed budgetary figures spread across a long cabinet table.",
				"Macro shot of a marble floor reflecting the silhouettes of passing staff.",
			},
			{
				"Close-up of engraved brass nameplates arranged along an inquiry dais.",
				"Macro shot of yellow-highlighted lines in a sworn congressional transcript.",
				"Close-up of an official government seal printed atop an investigative report.",
				"Medium shot of empty wooden jury or spectator seats inside a silent room.",
				"Close-up of a hand turning the thick page of a bound oversight dossier.",
				"Medium shot of stacked cardboard archive boxes inside a quiet records room.",
			},
			{
				"Close-up of an empty wooden desk surface lit by slanted morning sunlight.",
				"Medium shot of empty witness chairs facing a silent legislative dais.",
				"Close-up of printed public record totals in black ink on white paper.",
				"Medium shot of tall double mahogany doors closing into shadow.",
				"Close-up of empty spectator seats inside a quiet public gallery.",
				"Medium close-up of an unlit television monitor resting on a press stand.",
			},
		},
		CharacterEras: [3]CharacterEraTemplate{
			{
				Role:       "Main Subject — Early Public Service",
				AgeRange:   "Student, Clerk & Local Organizer Era",
				Appearance: "Earnest, observant demeanor in modest district offices and community halls.",
				Hair:       "Simple, neat early-career grooming.",
				Clothing:   "Modest off-the-rack blazer, rolled shirtsleeves, worn leather briefcase.",
				Body:       "Alert, forward-leaning posture over local maps and legal notebooks.",
				Locations:  "Law library, storefront office, local town hall.",
				Palette:    "Warm paper cream, muted navy, worn wood brown",
			},
			{
				Role:       "Main Subject — Rise to Office",
				AgeRange:   "National Leadership & Policy Era",
				Appearance: "Composed, commanding presence beneath press lights and legislative balconies.",
				Hair:       "Crisp, formal public-office grooming.",
				Clothing:   "Dark tailored suit, crisp collar, subtle lapel pin.",
				Body:       "Measured, upright stance behind podiums and diplomatic tables.",
				Locations:  "Capitol rotunda, press briefing room, executive chamber.",
				Palette:    "Marble white, deep slate, formal charcoal, muted brass",
			},
			{
				Role:       "Main Subject — Accountability & Historical Ledger",
				AgeRange:   "Inquiry & Later Statesperson Era",
				Appearance: "Weathered stillness under intense institutional and media scrutiny.",
				Hair:       "Silver-edged formal grooming in stark chamber lighting.",
				Clothing:   "Conservative dark suit, quiet tie, reading glasses beside heavy binders.",
				Body:       "Still, deliberate posture facing formal inquiry microphones.",
				Locations:  "Oversight committee room, federal courthouse corridor, quiet archive.",
				Palette:    "Mahogany, overcast stone grey, deep shadow contrast",
			},
		},
	}
}

func domainSportsCompetition() BiographicalDomainProfile {
	return BiographicalDomainProfile{
		Key:             "sports_competition",
		Label:           "Sports & Competition",
		OpeningRoom:     "a silent championship arena tunnel and press conference room lined with microphones",
		OriginRoom:      "a weathered neighborhood training hall lit by a single overhead bulb",
		ReckoningRoom:   "a federation review room, press dais, and empty center stage",
		CentralQuestion: "What happens when an overlooked challenger breaks open an entire sport—and what does the record say when the noise fades?",
		ClosingQuestion: "When a career is measured in tenths of a second and public expectation, what endures after the final whistle?",
		ChapterTitles: [7]string{
			"Cold Open",
			"The Early Discipline",
			"The First Upset",
			"Doubt & Disputes",
			"The Championship Stage",
			"Pressure & Reckoning",
			"The Open Question",
		},
		AmbientCues: []string{
			"Distant arena murmur continues",
			"Silence",
			"Quiet echo of footsteps in an empty training hall",
			"Quiet locker room ambient",
			"Ambient fadeout",
		},
		ChapterLocations: [7]string{
			"Championship press hall & quiet neighborhood training floor",
			"Hometown neighborhood street, local club & family home",
			"Regional tournament venue, travel bus & modest warm-up corridor",
			"Federation office, sports press desk & quiet locker room",
			"International stadium, center court & global broadcast booth",
			"Medical clinic, disciplinary/press hearing room & empty arena",
			"Empty hometown practice floor & quiet championship tunnel",
		},
		ChapterPrimaryShots: [7][]string{
			{
				"Wide shot of a packed international championship press room lined with broadcast microphones.",
				"Medium shot of an athlete seated calmly behind a press table under bright camera lights.",
				"Over-the-shoulder shot looking past the press dais toward rows of journalists and lenses.",
				"Wide exterior shot of a modest neighborhood training hall at dawn under streetlights.",
				"Interior shot of an empty local practice court lit by a single warm overhead fixture.",
				"Medium shot looking down a concrete stadium tunnel toward floodlit center stage.",
			},
			{
				"Wide shot of a quiet residential street in early morning light.",
				"Exterior shot of a modest family home with worn training gear resting near the doorway.",
				"Wide interior shot of a community gymnasium with scuffed hardwood floors and faded banners.",
				"Medium shot of chalk, tape, and a worn stopwatch laid out on a wooden bench.",
				"Exterior of a local school athletic field beneath an overcast sky.",
				"Wide shot of an empty neighborhood training room at twilight.",
			},
			{
				"Wide shot of a regional tournament hall with temporary bleachers and manual scoreboards.",
				"Low-angle shot looking up at bright stadium rafters from an empty warm-up floor.",
				"Medium shot of an official tournament bracket sheet pinned to a corkboard.",
				"Interior of a rain-streaked regional team bus parked outside a modest venue.",
				"Front page of a local sports newspaper resting on a wooden bench.",
				"Interior of a quiet hotel room with a travel bag and competition credentials on the desk.",
			},
			{
				"Printed federation notice and qualification standings inside a manila folder.",
				"Wide exterior shot of a national sports federation building under grey skies.",
				"Front-page sports editorial spread flat across a wooden table.",
				"Printed match or race telemetry sheets with marked time splits.",
				"Interior shot of an empty stadium locker room with rows of closed metal doors.",
				"View looking across empty spectator seats toward a dimly lit center floor.",
			},
			{
				"Aerial shot of a vast illuminated international stadium at night.",
				"Wide shot of center stage flooded with television broadcast lights.",
				"Medium shot of an electronic arena scoreboard displaying championship standings.",
				"Exterior view of an international broadcast compound lined with satellite trucks.",
				"Wide shot of spectators standing across tiered stands under bright flashbulbs.",
				"Interior of a quiet medal or trophy preparation corridor behind the main arena.",
			},
			{
				"Wide exterior shot of a sports medicine rehabilitation clinic at dusk.",
				"Formal federation press room with rows of microphones and blank sponsor boards.",
				"Printed injury report, contract filing, or federation statement resting on a wooden desk.",
				"Smartphone screen held in quiet evening light displaying rapid commentary feeds.",
				"Television monitor inside a quiet media room replaying archival tournament footage.",
				"Empty concrete tunnel leading back out to a darkened stadium floor.",
			},
			{
				"Wide shot of an empty hometown training hall bathed in clear morning window light.",
				"Wide shot of an empty championship arena with dark upper tiers.",
				"Official record books and press clippings resting side by side on a desk.",
				"Wide shot of an empty stadium corridor leading toward daylight outside.",
				"Over-the-shoulder view looking past an empty press table toward vacant seats.",
				"Final wide shot of a quiet, darkened practice floor with a single lit doorway.",
			},
		},
		ChapterCutawayShots: [7][]string{
			{
				"Close-up of telephoto camera lenses and silent shutters aimed at the dais.",
				"Macro shot of a tabletop press microphone catching quiet breathing.",
				"Close-up of taped hands resting calmly on a wooden bench.",
				"High-angle wide shot looking down over the entire arena floor from the rafters.",
				"Low-angle shot looking up at the weathered brick entrance of the local club.",
				"Macro shot of worn shoelaces being tied tightly before stepping onto the floor.",
			},
			{
				"Close-up of chalk dust lifting softly off weathered training equipment.",
				"Macro detail of a mechanical stopwatch ticking cleanly in hand.",
				"Close-up of handwritten training split times in a spiral notebook.",
				"Macro shot of scuffed court lines painted across aged hardwood.",
				"Close-up of a metal locker latch clicking shut in a quiet hall.",
				"Medium shot of an empty wooden gym bench beneath high frosted windows.",
			},
			{
				"Close-up of an unseeded entrant's name printed near the bottom of a draw sheet.",
				"Macro detail of a travel credential lanyard lying on a folding table.",
				"Close-up of a desk calendar turned to the exact weekend of the breakthrough upset.",
				"Macro shot of digital scoreboard digits clicking upward.",
				"Close-up of a pen circling a qualifying time on printed results.",
				"Macro detail of newsprint headlines folded beside a water bottle.",
			},
			{
				"Close-up of an official federation stamp printed atop a formal letter.",
				"Macro shot of underlined statistics inside a critical sports column.",
				"Close-up of an ice pack and athletic tape resting on an empty treatment table.",
				"Macro detail of worn scuff marks along an empty locker room floor.",
				"Close-up of a closed wooden door marked 'Officials Only'.",
				"High-angle shot of an empty center mark inside a silent venue.",
			},
			{
				"Close-up of championship credential badges lined neatly on a check-in desk.",
				"Macro shot of camera flash reflections rippling across glossy floor varnish.",
				"Close-up of a national emblem stitched cleanly onto competition fabric.",
				"Medium shot of illuminated stadium towers rising into the night sky.",
				"Close-up of printed world ranking charts spread across a broadcast desk.",
				"Macro shot of a gold medal ribbon resting motionless on dark felt.",
			},
			{
				"Close-up of engraved nameplates arranged along a press conference table.",
				"Macro shot of highlighted lines inside a federation disciplinary or medical summary.",
				"Close-up of a thumb scrolling through social media reaction posts on a phone.",
				"Medium shot of empty press rows inside a silent broadcast hall.",
				"Close-up of hands flipping through archived tournament programs.",
				"Medium shot of stacked archival tape cases inside a sports broadcast vault.",
			},
			{
				"Close-up of an empty wooden bench lit by slanted morning light.",
				"Medium shot of empty press chairs facing an unlit backdrop.",
				"Close-up of official record totals printed in black ink on white paper.",
				"Medium shot of tall arena tunnel doors opening onto a quiet street.",
				"Close-up of empty stadium seats receding into soft shadow.",
				"Medium close-up of an unlit broadcast monitor sitting beside the court.",
			},
		},
		CharacterEras: [3]CharacterEraTemplate{
			{
				Role:       "Main Subject — Formative Youth",
				AgeRange:   "Early Training & Local Club Era",
				Appearance: "Quiet, observant determination in modest community training rooms before national recognition.",
				Hair:       "Natural, neatly kept youth hair consistent with early archival photos.",
				Clothing:   "Simple worn training kit, modest local club jacket, scuffed athletic shoes.",
				Body:       "Lean, disciplined athletic build, coiled focus.",
				Locations:  "Neighborhood gym, weathered outdoor court, quiet family home.",
				Palette:    "Warm tungsten amber, dusty ochre, muted slate",
			},
			{
				Role:       "Main Subject — Contender & Crucible Era",
				AgeRange:   "Qualification & Breakthrough Era",
				Appearance: "Sharper jawline, intense focus under travel fatigue and international competition.",
				Hair:       "Consistent grooming slightly damp from match intensity.",
				Clothing:   "National travel tracksuit, competition credential lanyard, travel pack.",
				Body:       "Fully developed competitive posture, steady composure under pressure.",
				Locations:  "Regional tournament corridor, transit bus, quiet locker room.",
				Palette:    "Cool steel blue, stadium fluorescent, deep contrast shadow",
			},
			{
				Role:       "Main Subject — Championship & Public Record",
				AgeRange:   "Peak Recognition & Scrutiny Era",
				Appearance: "Measured calm under global press scrutiny and historic expectation.",
				Hair:       "Crisp press-day grooming under broadcast key lighting.",
				Clothing:   "Official finals competition kit or understated dark press jacket.",
				Body:       "Commanding, still presence on center stage and behind press microphones.",
				Locations:  "International stadium tunnel, broadcast press room, quiet training floor.",
				Palette:    "Championship gold, deep midnight navy, stark broadcast white",
			},
		},
	}
}

func domainArtsCulture() BiographicalDomainProfile {
	return BiographicalDomainProfile{
		Key:             "arts_culture",
		Label:           "Arts, Film & Music",
		OpeningRoom:     "a hushed theater auditorium and festival press hall illuminated by a single stage beam",
		OriginRoom:      "a cramped apartment worktable scattered with notebooks, tapes, and drafts",
		ReckoningRoom:   "a public critical retrospective, legal copyright room, and empty cinema hall",
		CentralQuestion: "What happens when one private artistic voice reshapes public culture—and where does the person end and the myth begin?",
		ClosingQuestion: "When an artist's work belongs to millions of strangers, who controls the record of the life behind it?",
		ChapterTitles: [7]string{
			"Cold Open",
			"Early Voices",
			"The First Work",
			"Critics & Ownership",
			"Cultural Scale",
			"Scrutiny & Reckoning",
			"The Open Question",
		},
		AmbientCues: []string{
			"Quiet murmur of the auditorium continues",
			"Silence",
			"Soft mechanical hum of a reel or studio monitor",
			"Quiet room tone",
			"Ambient fadeout",
		},
		ChapterLocations: [7]string{
			"Grand auditorium & cramped early studio",
			"Hometown neighborhood, rehearsal room & public library",
			"Small recording/editing room, indie theater & local press desk",
			"Publishing/label boardroom, courthouse & critical archive",
			"International festival premiere, concert hall & museum gallery",
			"Legal chamber, media press room & archival vault",
			"Empty early studio & quiet darkened auditorium",
		},
		ChapterPrimaryShots: [7][]string{
			{
				"Wide shot of a grand theater auditorium filled with press cameras and silent velvet rows.",
				"Medium shot of a single stage microphone standing under a focused overhead beam.",
				"Over-the-shoulder shot looking past the lectern toward dark tiers of audience seats.",
				"Wide exterior shot of a modest brick apartment building at night under streetlights.",
				"Interior shot of a narrow studio desk lit by a single warm lamp over handwritten pages.",
				"Medium shot looking down a quiet backstage corridor toward closed stage doors.",
			},
			{
				"Wide shot of a quiet neighborhood street in soft early morning light.",
				"Interior of a family living room with worn vinyl sleeves, books, or an upright instrument.",
				"Wide interior shot of a school rehearsal room with scuffed floors and music stands.",
				"Medium shot of handwritten drafts, sketches, or sheet music spread across a wooden table.",
				"Exterior of an older brick conservatory or public arts school under an overcast sky.",
				"Wide shot of a quiet neighborhood cinema or bookstore facade at twilight.",
			},
			{
				"Wide shot of a cramped studio room filled with cables, notes, and a glowing monitor.",
				"Low-angle shot of an illuminated editing bay or recording console in a dark room.",
				"Medium shot of a printed first-edition script, score, or exhibition catalog on a table.",
				"Wide exterior shot of a small independent venue marquee lit up at night.",
				"Front page of an arts review section spread flat across a wooden desk.",
				"Interior of a quiet coffeehouse corner with an open notebook and cooling cup.",
			},
			{
				"Formal publishing or rights contract inside a manila folder on a mahogany desk.",
				"Wide exterior shot of a courthouse or entertainment district headquarters under grey skies.",
				"Printed critical essays and competing interviews laid side by side under lamp light.",
				"Legal dispute filing lying open with highlighted clauses on white bond paper.",
				"Interior shot of an empty boardroom overlooking city rooftops.",
				"View looking across empty gallery benches toward an unlit exhibition wall.",
			},
			{
				"Aerial shot of a major cultural capital illuminated at twilight.",
				"Wide shot of a packed international concert hall or premiere theater under warm lights.",
				"Medium shot of a cinema marquee glowing brightly along a wet city avenue.",
				"Exterior view of a major museum gallery entrance lined with vertical banners.",
				"Wide shot of an audience holding lit programs inside a hushed auditorium.",
				"Interior of a climate-controlled archival vault lined with labeled master reels.",
			},
			{
				"Wide exterior shot of a courthouse or public arts institution under overcast skies.",
				"Press conference room lined with microphones and camera tripods.",
				"Printed legal filings, critical retrospectives, and formal statements on a conference table.",
				"Smartphone screen in evening light displaying rapid cultural debate threads.",
				"Television monitor inside a quiet screening room playing archived interview footage.",
				"Stack of numbered archival folders resting along steel shelving.",
			},
			{
				"Wide shot of an empty early studio desk bathed in clear morning window light.",
				"Wide shot of an empty theater auditorium with rows of folded crimson seats.",
				"Published works and critical dossiers resting side by side on dark wood.",
				"Wide shot of an empty marble gallery hallway leading to closed doors.",
				"Over-the-shoulder view looking past an empty stage toward silent balconies.",
				"Final wide shot of a dark, quiet auditorium with a single beam of light.",
			},
		},
		ChapterCutawayShots: [7][]string{
			{
				"Close-up of camera lenses catching quiet stage light from the press pit.",
				"Macro shot of a vintage studio microphone grille in soft side lighting.",
				"Close-up of folded hands resting on a velvet armrest in the dark.",
				"High-angle wide shot looking down over the entire auditorium from the balcony.",
				"Low-angle shot looking up at the brick facade of the early apartment building.",
				"Macro shot of a fountain pen moving steadily across unlined paper.",
			},
			{
				"Close-up of worn notebook margins covered in early handwritten revisions.",
				"Macro detail of a needle resting quietly above a vinyl record groove.",
				"Close-up of pencil shavings beside a sketchpad under warm lamp light.",
				"Macro shot of weathered book spines lined along a bedroom shelf.",
				"Close-up of rain streaks sliding down a rehearsal room window.",
				"Medium shot of an empty wooden stool standing beside a music stand.",
			},
			{
				"Close-up of a desk calendar circled on the exact release or premiere date.",
				"Macro detail of glowing level meters pulsing inside a quiet studio.",
				"Close-up of red ink editorial notes across a manuscript page.",
				"Macro shot of glowing bulbs along an independent theater marquee.",
				"Close-up of printed review text clipped cleanly from newsprint.",
				"Medium shot of a closed studio door with an unlit 'On Air' sign.",
			},
			{
				"Close-up of a red wax or notary seal stamped onto a rights agreement.",
				"Macro shot of contract royalty lines highlighted in yellow ink.",
				"Close-up of reading glasses resting beside an open biography.",
				"Macro detail of redacted clauses across a settlement document.",
				"Close-up of a wooden gavel or brass paperweight on a legal desk.",
				"High-angle shot of an empty negotiation table inside a glass room.",
			},
			{
				"Close-up of an award statuette silhouette resting in quiet shadow.",
				"Macro shot of printed ticket stubs stacked beside a premiere program.",
				"Close-up of spotlight filaments warming above an empty stage.",
				"Medium shot of evening crowds passing beneath illuminated festival banners.",
				"Close-up of printed sales and box-office tallies on white paper.",
				"Macro shot of archival tape labels written in neat black ink.",
			},
			{
				"Close-up of engraved nameplates along a panel or inquiry table.",
				"Macro shot of highlighted paragraphs inside a retrospective essay.",
				"Close-up of a thumb scrolling steadily through online commentary.",
				"Medium shot of empty screening room chairs facing a dark screen.",
				"Close-up of hands turning the pages of a thick archival scrapbook.",
				"Medium shot of stacked cardboard archive boxes inside a quiet vault.",
			},
			{
				"Close-up of an empty wooden worktable lit by slanted morning sun.",
				"Medium shot of empty auditorium seats receding into quiet shadow.",
				"Close-up of printed critical record summaries on white paper.",
				"Medium shot of tall backstage doors closing softly.",
				"Close-up of an unlit projector lens inside a silent booth.",
				"Medium close-up of an unlit studio monitor resting on a wooden stand.",
			},
		},
		CharacterEras: [3]CharacterEraTemplate{
			{
				Role:       "Main Subject — Early Creative Voice",
				AgeRange:   "Apprenticeship & First Works Era",
				Appearance: "Introspective, intense focus inside modest studios and rehearsal rooms.",
				Hair:       "Natural, unstudied period-accurate hair.",
				Clothing:   "Worn cotton jacket, plain knit sweater, ink-stained notebook in hand.",
				Body:       "Still, observant posture bent over drafts, instruments, or editing decks.",
				Locations:  "Small apartment worktable, local rehearsal hall, independent venue.",
				Palette:    "Warm tungsten amber, film noir charcoal, muted sepia",
			},
			{
				Role:       "Main Subject — Breakthrough & Acclaim",
				AgeRange:   "Major Works & Cultural Prominence Era",
				Appearance: "Composed, enigmatic public presence amid festival lights and critical debate.",
				Hair:       "Distinctive, well-kept signature styling.",
				Clothing:   "Tailored minimalist dark coat or structured evening jacket.",
				Body:       "Measured poise on festival stages and inside recording/editing studios.",
				Locations:  "International cinema premiere, concert hall, museum gallery.",
				Palette:    "Deep velvet crimson, warm spotlight gold, stark monochrome",
			},
			{
				Role:       "Main Subject — Retrospective & Legacy",
				AgeRange:   "Archival Reflection & Cultural Record Era",
				Appearance: "Quiet authority shaped by decades of acclaim, dispute, and reinvention.",
				Hair:       "Refined mature grooming under soft natural window light.",
				Clothing:   "Understated dark wool coat, crisp dark shirt, timeless eyewear.",
				Body:       "Calm, grounded presence inside quiet studios and archival halls.",
				Locations:  "Quiet studio archive, empty auditorium, sunlit reading room.",
				Palette:    "Slate silver, warm wood, restrained documentary contrast",
			},
		},
	}
}

func domainHumanitarianCivic() BiographicalDomainProfile {
	p := domainPoliticsLaw()
	p.Key = "humanitarian_civic"
	p.Label = "Humanitarian & Civic Impact"
	p.OpeningRoom = "a quiet international assembly hall lined with translation headsets and press cameras"
	p.OriginRoom = "a modest neighborhood community office with hand-lettered charts on the wall"
	p.ReckoningRoom = "a public oversight hearing and a quiet field clinic archive"
	p.CentralQuestion = "What happens when one person builds a movement outside established institutions—and how does the record measure what endures?"
	p.ClosingQuestion = "When moral urgency meets institutional reality, what remains after the speeches end?"
	p.ChapterTitles = [7]string{
		"Cold Open",
		"Early Witness",
		"The First Initiative",
		"Resistance & Disputes",
		"Global Reach",
		"Scrutiny & Reckoning",
		"The Open Question",
	}
	p.ChapterLocations = [7]string{
		"International assembly hall & local community room",
		"Hometown neighborhood, local school & family table",
		"Grassroots clinic/office, municipal hall & regional road",
		"Oversight room, newsroom archive & local courthouse",
		"International summit, regional relief hub & university hall",
		"Audit hearing chamber, field archive & press room",
		"Empty neighborhood office & quiet assembly hall",
	}
	return p
}

func domainGeneralBiography() BiographicalDomainProfile {
	p := domainPoliticsLaw()
	p.Key = "general_biography"
	p.Label = "Investigative Biography"
	p.OpeningRoom = "a quiet public auditorium lined with press microphones and wooden seats"
	p.OriginRoom = "a modest early study lit by a single desk lamp"
	p.ReckoningRoom = "a public inquiry room and an institutional archive"
	p.CentralQuestion = "What happens when one individual's decisions ripple across millions of lives—and what does the verified record leave behind?"
	p.ClosingQuestion = "Between public myth and documented history, what answers remain open?"
	p.ChapterTitles = [7]string{
		"Cold Open",
		"Origins",
		"The Breakthrough",
		"The Contested Record",
		"Scale & Influence",
		"Reckoning",
		"The Open Question",
	}
	return p
}

// EnforceMaxSevenSecondPacing guarantees:
// 1. Every single spoken shot is <= 17 words and <= 7.0 seconds (splitting long lines with `...` bridges
//    and inserting cutaway visuals).
// 2. Every single shot (spoken or ambient) has 3.0 <= DurationSeconds <= 7.0 (NEVER > 7.0).
// 3. Every chapter fills its allocated time window cleanly so the documentary hits targetDurationMinutes
//    with continuous cumulative [M:SS] timestamps advancing by <= 7 seconds per cut.
func EnforceMaxSevenSecondPacing(
	sections []ScriptSection,
	domain BiographicalDomainProfile,
	input BiographyInput,
	targetDurationMinutes int,
) []ScriptSection {
	if targetDurationMinutes < 15 || targetDurationMinutes > 25 {
		targetDurationMinutes = 20
	}
	totalSeconds := targetDurationMinutes * 60
	windows := ai.ComputeChapterTimeWindows(totalSeconds)

	// Prepare supplemental factual beats from biography/claims so padding uses authentic substance first
	supplementalFacts := collectUniqueBioSentences(input)
	suppCursor := 0

	globalShotNum := 1
	for chIdx := range sections {
		w := [2]int{0, totalSeconds}
		if chIdx < len(windows) {
			w = windows[chIdx]
		}
		targetChDur := float64(w[1] - w[0])
		if targetChDur < 60 {
			targetChDur = 90
		}
		domChIdx := chIdx % 7
		loc := domain.ChapterLocations[domChIdx]

		// Step 1: Split any shot with > 17 words or > 7.0s duration into <= 7.0s shots with fresh cutaway visuals
		var normalized []ScriptShot
		cutawayCounter := 0
		for _, sh := range sections[chIdx].Shots {
			sh.Narration = strings.TrimSpace(sh.Narration)
			sh.Visual = ensureConcreteVisual(sh.Visual, domain, domChIdx, cutawayCounter, cutawayCounter%2 == 1)
			if strings.TrimSpace(sh.LocationRef) == "" {
				sh.LocationRef = loc
			}

			if sh.Narration == "" {
				cue := strings.TrimSpace(sh.AudioCue)
				if cue == "" {
					cue = "Silence"
				}
				dur := sh.DurationSeconds
				if dur <= 0 {
					dur = 6.0
				}
				for dur > MaxShotDurationSeconds {
					normalized = append(normalized, ScriptShot{
						Visual:          sh.Visual,
						Narration:       "",
						AudioCue:        cue,
						DurationSeconds: MaxShotDurationSeconds,
						LocationRef:     sh.LocationRef,
						ChapterNumber:   sections[chIdx].ChapterNumber,
						ChapterTitle:    sections[chIdx].Title,
					})
					dur -= MaxShotDurationSeconds
					cutawayCounter++
					sh.Visual = pickShotVisual(domain, domChIdx, cutawayCounter, true)
				}
				if dur < MinShotDurationSeconds {
					dur = 4.0
				}
				normalized = append(normalized, ScriptShot{
					Visual:          sh.Visual,
					Narration:       "",
					AudioCue:        cue,
					DurationSeconds: math.Round(dur),
					LocationRef:     sh.LocationRef,
					ChapterNumber:   sections[chIdx].ChapterNumber,
					ChapterTitle:    sections[chIdx].Title,
				})
				cutawayCounter++
				continue
			}

			chunks := splitNarrationToSevenSecondChunks(sh.Narration, MaxWordsPerShot)
			for ci, chunk := range chunks {
				vis := sh.Visual
				if ci > 0 {
					cutawayCounter++
					vis = pickShotVisual(domain, domChIdx, cutawayCounter, true)
				}
				dur := estimateShotSeconds(chunk, sh.DurationSeconds, len(chunks) > 1)
				normalized = append(normalized, ScriptShot{
					Visual:          vis,
					Narration:       chunk,
					AudioCue:        "",
					DurationSeconds: dur,
					LocationRef:     sh.LocationRef,
					ChapterNumber:   sections[chIdx].ChapterNumber,
					ChapterTitle:    sections[chIdx].Title,
				})
				cutawayCounter++
			}
		}

		// Step 2: Calculate current chapter duration and align to targetChDur without ever exceeding 7.0s per shot
		sumDur := func(list []ScriptShot) float64 {
			t := 0.0
			for _, s := range list {
				t += s.DurationSeconds
			}
			return t
		}

		// If chapter is under target duration, first bring spoken shots gently up to 7s max (still <= 7s),
		// then insert authentic archival/detail narration & atmospheric cutaway beats until filled.
		curDur := sumDur(normalized)
		if curDur < targetChDur {
			for i := range normalized {
				if curDur >= targetChDur {
					break
				}
				if normalized[i].DurationSeconds < MaxShotDurationSeconds {
					add := math.Min(MaxShotDurationSeconds-normalized[i].DurationSeconds, targetChDur-curDur)
					normalized[i].DurationSeconds = math.Round(normalized[i].DurationSeconds + add)
					curDur = sumDur(normalized)
				}
			}
		}

		// Insert additional sourced detail shots & atmospheric breathing cuts until curDur reaches targetChDur
		insertCycle := 0
		for curDur+MinShotDurationSeconds <= targetChDur {
			rem := targetChDur - curDur
			stepDur := math.Min(MaxShotDurationSeconds, math.Max(MinShotDurationSeconds, math.Round(rem)))
			if rem > MaxShotDurationSeconds && rem < 10.0 {
				stepDur = math.Floor(rem / 2.0)
				if stepDur < MinShotDurationSeconds {
					stepDur = MinShotDurationSeconds
				}
			}

			if insertCycle%3 != 2 && len(supplementalFacts) > 0 && rem >= 4.0 {
				fact := supplementalFacts[suppCursor%len(supplementalFacts)]
				suppCursor++
				subChunks := splitNarrationToSevenSecondChunks(fact, MaxWordsPerShot)
				addedAny := false
				for si, sc := range subChunks {
					remNow := targetChDur - curDur
					if remNow < MinShotDurationSeconds {
						break
					}
					d := math.Min(MaxShotDurationSeconds, math.Min(estimateShotSeconds(sc, 7.0, false), remNow))
					if d < MinShotDurationSeconds {
						d = MinShotDurationSeconds
					}
					if remNow-d > 0 && remNow-d < MinShotDurationSeconds {
						// Avoid leaving an unfillable 1-2s remainder
						if remNow <= MaxShotDurationSeconds {
							d = remNow
						} else {
							d = math.Floor(remNow / 2.0)
						}
					}
					vis := pickShotVisual(domain, domChIdx, cutawayCounter+insertCycle+si, si > 0 || insertCycle%2 == 1)
					s := ScriptShot{
						Visual:          vis,
						Narration:       sc,
						LocationRef:     loc,
						ChapterNumber:   sections[chIdx].ChapterNumber,
						ChapterTitle:    sections[chIdx].Title,
						DurationSeconds: math.Round(d),
					}
					normalized = insertBeforeClosingBeats(normalized, s)
					curDur = sumDur(normalized)
					addedAny = true
				}
				if addedAny {
					insertCycle++
					continue
				}
			}

			cue := domain.AmbientCues[(insertCycle+chIdx)%len(domain.AmbientCues)]
			vis := pickShotVisual(domain, domChIdx, cutawayCounter+insertCycle, true)
			newShot := ScriptShot{
				Visual:          vis,
				Narration:       "",
				AudioCue:        cue,
				LocationRef:     loc,
				ChapterNumber:   sections[chIdx].ChapterNumber,
				ChapterTitle:    sections[chIdx].Title,
				DurationSeconds: math.Round(stepDur),
			}
			normalized = append(normalized, newShot)
			curDur = sumDur(normalized)
			insertCycle++
			if insertCycle > 140 {
				break
			}
		}

		// Close any tiny 1-2s remainder so chapter total matches targetChDur exactly without exceeding 7s/cut
		for curDur < targetChDur {
			adjusted := false
			for i := range normalized {
				if curDur >= targetChDur {
					break
				}
				if normalized[i].DurationSeconds < MaxShotDurationSeconds {
					normalized[i].DurationSeconds += 1.0
					curDur += 1.0
					adjusted = true
				}
			}
			if !adjusted {
				// All shots are already at 7.0s and 1-2s remain: borrow 2s from last shot and add a 3-4s closing cutaway
				if len(normalized) > 0 {
					rem := targetChDur - curDur
					normalized[len(normalized)-1].DurationSeconds -= 2.0
					normalized = append(normalized, ScriptShot{
						Visual:          pickShotVisual(domain, domChIdx, cutawayCounter+len(normalized), true),
						Narration:       "",
						AudioCue:        "Silence",
						LocationRef:     loc,
						ChapterNumber:   sections[chIdx].ChapterNumber,
						ChapterTitle:    sections[chIdx].Title,
						DurationSeconds: 2.0 + rem,
					})
					curDur = sumDur(normalized)
				}
				break
			}
		}

		// If chapter exceeds targetChDur, tighten shot durations down to >= 3.0s so sum == targetChDur exactly
		if curDur > targetChDur {
			excess := curDur - targetChDur
			for i := len(normalized) - 1; i >= 0 && excess > 0; i-- {
				if normalized[i].DurationSeconds > MinShotDurationSeconds {
					reducible := math.Min(normalized[i].DurationSeconds-MinShotDurationSeconds, excess)
					normalized[i].DurationSeconds = math.Round(normalized[i].DurationSeconds - reducible)
					excess -= reducible
				}
			}
			// If still over budget after tightening all shots to 3s, drop trailing ambient padding shots
			for excess > 0 && len(normalized) > 4 {
				last := normalized[len(normalized)-1]
				normalized = normalized[:len(normalized)-1]
				excess -= last.DurationSeconds
			}
			// Re-top up if dropping a shot went slightly under
			curDur = sumDur(normalized)
			for i := 0; i < len(normalized) && curDur < targetChDur; i++ {
				if normalized[i].DurationSeconds < MaxShotDurationSeconds {
					add := math.Min(MaxShotDurationSeconds-normalized[i].DurationSeconds, targetChDur-curDur)
					normalized[i].DurationSeconds += add
					curDur += add
				}
			}
		}

		// Step 3: Assign sequential shot numbers and exact cumulative [M:SS] timestamps inside chapter window
		cursorSec := float64(w[0])
		var paragraphs []string
		for i := range normalized {
			if normalized[i].DurationSeconds > MaxShotDurationSeconds {
				normalized[i].DurationSeconds = MaxShotDurationSeconds
			}
			if normalized[i].DurationSeconds < MinShotDurationSeconds {
				normalized[i].DurationSeconds = MinShotDurationSeconds
			}
			normalized[i].ShotNumber = globalShotNum
			globalShotNum++
			normalized[i].StartSeconds = cursorSec
			normalized[i].EndSeconds = cursorSec + normalized[i].DurationSeconds
			normalized[i].Timestamp = formatMMSS(int(math.Round(cursorSec)))
			cursorSec = normalized[i].EndSeconds

			if normalized[i].Narration != "" {
				paragraphs = append(paragraphs, normalized[i].Narration)
			}
		}

		sections[chIdx].Shots = normalized
		sections[chIdx].Paragraphs = paragraphs
		sections[chIdx].TimeRange = fmt.Sprintf("%s – %s", formatMMSS(w[0]), formatMMSS(w[1]))
		if sections[chIdx].DomainField == "" {
			sections[chIdx].DomainField = domain.Label
		}
		if sections[chIdx].CentralQuestion == "" {
			sections[chIdx].CentralQuestion = domain.CentralQuestion
		}
	}

	return sections
}

func insertBeforeClosingBeats(list []ScriptShot, item ScriptShot) []ScriptShot {
	if len(list) <= 2 {
		return append(list, item)
	}
	idx := len(list) - 2
	out := make([]ScriptShot, 0, len(list)+1)
	out = append(out, list[:idx]...)
	out = append(out, item)
	out = append(out, list[idx:]...)
	return out
}

// FormatRealignedScript renders the complete shot-by-shot documentary script in clean
// side-by-side VISUAL / NARRATOR (V.O.) / (Audio: ...) format at <= 7 seconds per cut.
func FormatRealignedScript(sections []ScriptSection) string {
	var sb strings.Builder

	for idx, sec := range sections {
		if idx > 0 {
			sb.WriteString("\n---\n\n")
		}
		header := strings.TrimSpace(sec.Title)
		if !strings.Contains(header, "·") {
			header = fmt.Sprintf("%d · %s", sec.ChapterNumber, header)
		}
		if sec.TimeRange != "" {
			sb.WriteString(fmt.Sprintf("%s (%s)\n\n", header, sec.TimeRange))
		} else {
			sb.WriteString(header + "\n\n")
		}

		for _, sh := range sec.Shots {
			sb.WriteString(fmt.Sprintf("[%s] VISUAL: %s  \n", sh.Timestamp, strings.TrimSpace(sh.Visual)))
			if strings.TrimSpace(sh.Narration) != "" {
				sb.WriteString(fmt.Sprintf("NARRATOR (V.O.): %s\n\n", strings.TrimSpace(sh.Narration)))
			} else {
				cue := strings.TrimSpace(sh.AudioCue)
				if cue == "" {
					cue = "Silence"
				}
				sb.WriteString(fmt.Sprintf("(Audio: %s)\n\n", cue))
			}
		}
	}

	return strings.TrimSpace(sb.String()) + "\n"
}

var (
	reShotHeader  = regexp.MustCompile(`^\[(\d+:\d{2})\]\s+VISUAL(?:\s+\[[^\]]+\])?:\s*(.+)$`)
	reChapterHead = regexp.MustCompile(`^(\d+)\s*[·\-]\s*(.+?)(?:\s*\(([^)]+)\))?$`)
	reAudioCue    = regexp.MustCompile(`^\(Audio:\s*(.+?)\)$`)
)

// ExtractSpokenNarration returns ONLY the clean NARRATOR (V.O.) spoken lines from a realigned script
// (or the original prose if the script has no visual shot markers).
func ExtractSpokenNarration(fullScript string) string {
	lines := strings.Split(fullScript, "\n")
	var spoken []string
	hasMarkers := false
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "NARRATOR (V.O.):") {
			hasMarkers = true
			txt := strings.TrimSpace(strings.TrimPrefix(line, "NARRATOR (V.O.):"))
			if txt != "" {
				spoken = append(spoken, txt)
			}
		}
	}
	if hasMarkers && len(spoken) > 0 {
		return strings.Join(spoken, "\n\n")
	}
	return strings.TrimSpace(fullScript)
}

// ParseRealignedScriptShots parses a realigned documentary script text back into structured shots,
// enforcing <= 7.0s per cut so manual producer edits sync directly into the visual timeline.
func ParseRealignedScriptShots(fullScript string, domain BiographicalDomainProfile) []ScriptShot {
	lines := strings.Split(fullScript, "\n")
	var shots []ScriptShot
	currentChNum := 0
	currentChTitle := "0 · Cold Open"

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || line == "---" {
			continue
		}
		if m := reChapterHead.FindStringSubmatch(line); len(m) >= 3 && !strings.HasPrefix(line, "[") {
			n, _ := strconv.Atoi(m[1])
			currentChNum = n
			currentChTitle = fmt.Sprintf("%d · %s", n, strings.TrimSpace(m[2]))
			continue
		}
		if m := reShotHeader.FindStringSubmatch(line); len(m) == 3 {
			tsStr := m[1]
			vis := strings.TrimSpace(m[2])
			startSec := parseMMSS(tsStr)

			narration := ""
			audioCue := ""
			// Look ahead to the next non-empty line for NARRATOR (V.O.) or (Audio: ...)
			for j := i + 1; j < len(lines); j++ {
				nextLine := strings.TrimSpace(lines[j])
				if nextLine == "" {
					continue
				}
				if strings.HasPrefix(nextLine, "NARRATOR (V.O.):") {
					narration = strings.TrimSpace(strings.TrimPrefix(nextLine, "NARRATOR (V.O.):"))
					i = j
				} else if am := reAudioCue.FindStringSubmatch(nextLine); len(am) == 2 {
					audioCue = strings.TrimSpace(am[1])
					i = j
				}
				break
			}

			loc := domain.ChapterLocations[currentChNum%7]
			shots = append(shots, ScriptShot{
				Timestamp:     tsStr,
				StartSeconds:  float64(startSec),
				Visual:        vis,
				Narration:     narration,
				AudioCue:      audioCue,
				LocationRef:   loc,
				ChapterNumber: currentChNum,
				ChapterTitle:  currentChTitle,
			})
		}
	}

	if len(shots) == 0 {
		return nil
	}

	// Compute durations from adjacent timestamps and enforce <= 7.0s per cut
	var clamped []ScriptShot
	cutCounter := 0
	for i, sh := range shots {
		var dur float64
		if i+1 < len(shots) && shots[i+1].StartSeconds > sh.StartSeconds {
			dur = shots[i+1].StartSeconds - sh.StartSeconds
		} else {
			dur = estimateShotSeconds(sh.Narration, 6.0, false)
		}
		if dur < MinShotDurationSeconds {
			dur = 4.0
		}

		// Split if narration > 17 words or duration > 7.0s
		if sh.Narration != "" && (len(strings.Fields(sh.Narration)) > MaxWordsPerShot || dur > MaxShotDurationSeconds) {
			chunks := splitNarrationToSevenSecondChunks(sh.Narration, MaxWordsPerShot)
			for ci, chunk := range chunks {
				vis := sh.Visual
				if ci > 0 {
					cutCounter++
					vis = pickShotVisual(domain, sh.ChapterNumber%7, cutCounter, true)
				}
				clamped = append(clamped, ScriptShot{
					Visual:          vis,
					Narration:       chunk,
					DurationSeconds: estimateShotSeconds(chunk, 7.0, len(chunks) > 1),
					LocationRef:     sh.LocationRef,
					ChapterNumber:   sh.ChapterNumber,
					ChapterTitle:    sh.ChapterTitle,
				})
			}
			continue
		}

		for dur > MaxShotDurationSeconds {
			clamped = append(clamped, ScriptShot{
				Visual:          sh.Visual,
				Narration:       sh.Narration,
				AudioCue:        sh.AudioCue,
				DurationSeconds: MaxShotDurationSeconds,
				LocationRef:     sh.LocationRef,
				ChapterNumber:   sh.ChapterNumber,
				ChapterTitle:    sh.ChapterTitle,
			})
			dur -= MaxShotDurationSeconds
			cutCounter++
			sh.Visual = pickShotVisual(domain, sh.ChapterNumber%7, cutCounter, true)
			sh.Narration = ""
			if sh.AudioCue == "" {
				sh.AudioCue = "Room ambient"
			}
		}
		if dur < MinShotDurationSeconds {
			dur = 4.0
		}
		sh.DurationSeconds = math.Round(dur)
		clamped = append(clamped, sh)
	}

	// Recompute exact sequential timestamps
	cur := 0.0
	for i := range clamped {
		clamped[i].ShotNumber = i + 1
		clamped[i].StartSeconds = cur
		clamped[i].EndSeconds = cur + clamped[i].DurationSeconds
		clamped[i].Timestamp = formatMMSS(int(math.Round(cur)))
		cur = clamped[i].EndSeconds
	}
	return clamped
}

func splitNarrationToSevenSecondChunks(text string, maxWords int) []string {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	if len(words) <= maxWords {
		return []string{strings.Join(words, " ")}
	}

	// Calculate how many balanced chunks we need so every chunk <= maxWords
	numChunks := (len(words) + maxWords - 1) / maxWords
	targetPerChunk := (len(words) + numChunks - 1) / numChunks
	if targetPerChunk > maxWords {
		targetPerChunk = maxWords
	}

	var chunks []string
	start := 0
	for start < len(words) {
		remaining := len(words) - start
		if remaining <= maxWords {
			chunk := strings.Join(words[start:], " ")
			if len(chunks) > 0 && !strings.HasPrefix(chunk, "...") && !isCapitalizedSentenceStart(chunks[len(chunks)-1]) {
				chunk = "..." + chunk
			}
			chunks = append(chunks, chunk)
			break
		}

		// Find best split point near targetPerChunk (prefer sentence end, then comma/semicolon/colon, then conjunction)
		end := start + targetPerChunk
		if end > start+maxWords {
			end = start + maxWords
		}
		bestSplit := end
		minSplit := start + max(6, targetPerChunk-5)
		if minSplit >= end {
			minSplit = start + 6
		}
		if minSplit > len(words)-3 {
			minSplit = start + 1
		}

		// Pass 1: Look for sentence boundary (. ? !)
		found := false
		for idx := end; idx >= minSplit; idx-- {
			w := words[idx-1]
			if strings.HasSuffix(w, ".") || strings.HasSuffix(w, "?") || strings.HasSuffix(w, "!") {
				bestSplit = idx
				found = true
				break
			}
		}
		// Pass 2: Look for clause punctuation (, ; : —)
		if !found {
			for idx := end; idx >= minSplit; idx-- {
				w := words[idx-1]
				if strings.HasSuffix(w, ",") || strings.HasSuffix(w, ";") || strings.HasSuffix(w, ":") || strings.HasSuffix(w, "—") {
					bestSplit = idx
					found = true
					break
				}
			}
		}
		// Pass 3: Look for natural conjunction
		if !found {
			for idx := end; idx >= minSplit; idx-- {
				lw := strings.ToLower(words[idx])
				if lw == "and" || lw == "but" || lw == "while" || lw == "where" || lw == "which" || lw == "after" || lw == "before" || lw == "with" {
					bestSplit = idx
					break
				}
			}
		}

		part := strings.Join(words[start:bestSplit], " ")
		endsSentence := strings.HasSuffix(part, ".") || strings.HasSuffix(part, "?") || strings.HasSuffix(part, "!")
		if len(chunks) > 0 && !strings.HasPrefix(part, "...") && !isCapitalizedSentenceStart(chunks[len(chunks)-1]) {
			part = "..." + part
		}
		if !endsSentence && bestSplit < len(words) {
			part = strings.TrimRight(part, ",;:") + "..."
		}
		chunks = append(chunks, part)
		start = bestSplit
	}

	return chunks
}

func isCapitalizedSentenceStart(prevChunk string) bool {
	prev := strings.TrimSpace(prevChunk)
	if strings.HasSuffix(prev, "...") {
		return false
	}
	return strings.HasSuffix(prev, ".") || strings.HasSuffix(prev, "?") || strings.HasSuffix(prev, "!")
}

func estimateShotSeconds(narration string, hint float64, wasSplit bool) float64 {
	words := len(strings.Fields(narration))
	if words == 0 {
		if hint >= MinShotDurationSeconds && hint <= MaxShotDurationSeconds {
			return math.Round(hint)
		}
		return 6.0
	}
	est := math.Ceil(float64(words) / WordsPerSecondTarget)
	if !wasSplit && hint >= MinShotDurationSeconds && hint <= MaxShotDurationSeconds && math.Abs(hint-est) <= 2.0 {
		est = math.Round(hint)
	}
	if est < 4.0 {
		est = 5.0
	}
	if est > MaxShotDurationSeconds {
		est = MaxShotDurationSeconds
	}
	return est
}

func pickShotVisual(domain BiographicalDomainProfile, chapterIdx int, counter int, cutaway bool) string {
	ch := chapterIdx % 7
	if cutaway {
		pool := domain.ChapterCutawayShots[ch]
		if len(pool) > 0 {
			return pool[counter%len(pool)]
		}
	}
	pool := domain.ChapterPrimaryShots[ch]
	if len(pool) > 0 {
		return pool[counter%len(pool)]
	}
	return "Medium shot of an archival documentary workspace lit by natural window light."
}

func ensureConcreteVisual(vis string, domain BiographicalDomainProfile, chIdx int, counter int, cutaway bool) string {
	vis = strings.TrimSpace(vis)
	if len(vis) < 16 {
		return pickShotVisual(domain, chIdx, counter, cutaway)
	}
	return vis
}

func collectUniqueBioSentences(input BiographyInput) []string {
	seen := map[string]bool{}
	var out []string
	addBlock := func(raw string) {
		for _, s := range splitSentences(raw) {
			s = strings.TrimSpace(s)
			if len(s) < 24 {
				continue
			}
			key := strings.ToLower(s)
			if !seen[key] {
				seen[key] = true
				out = append(out, s)
			}
		}
	}
	addBlock(input.Biography)
	for _, c := range input.Claims {
		addBlock(c.Text)
	}
	for _, t := range input.TimelineEntries {
		addBlock(t)
	}
	addBlock(input.Summary)
	return out
}

func distributeFactsAcrossChapters(facts []string, input BiographyInput) [7][]string {
	var buckets [7][]string
	if len(facts) == 0 {
		return buckets
	}

	// Route sentences dynamically by biographical phase cues, falling back to chronological order
	for idx, f := range facts {
		lf := strings.ToLower(f)
		assigned := -1
		switch {
		case containsAny(lf, []string{"born", "childhood", "grew up", "parents", "father", "mother", "high school", "academy", "graduated", "studied", "early life", "university"}):
			assigned = 1
		case containsAny(lf, []string{"sued", "lawsuit", "accused", "dispute", "settled", "counter", "alleged", "critic", "controversy", "dismissed", "rejected", "failure", "defeat", "rival", "challenged"}):
			if idx > len(facts)/2 {
				assigned = 5
			} else {
				assigned = 3
			}
		case containsAny(lf, []string{"testified", "senate", "congress", "hearing", "trial", "verdict", "jury", "penalty", "subpoena", "investigation", "scandal", "inquiry", "regulator", "politics", "regret", "apologized"}):
			assigned = 5
		case containsAny(lf, []string{"million", "billion", "global", "world", "award", "prize", "founded", "ipo", "public", "pledge", "philanthrop", "expanded", "record", "champion", "milestone"}):
			assigned = 4
		case containsAny(lf, []string{"first", "launched", "created", "built", "debut", "breakthrough", "invented", "published", "prototype", "discovered"}):
			assigned = 2
		}

		if assigned == -1 {
			// Chronological fallback across chapters 1..5
			progress := float64(idx) / math.Max(1, float64(len(facts)))
			switch {
			case progress < 0.20:
				assigned = 1
			case progress < 0.40:
				assigned = 2
			case progress < 0.58:
				assigned = 3
			case progress < 0.78:
				assigned = 4
			default:
				assigned = 5
			}
		}
		buckets[assigned] = append(buckets[assigned], f)
	}

	return buckets
}

func containsAny(s string, needles []string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func cleanShortSentence(s string, maxWords int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if s == "" {
		return ""
	}
	sents := splitSentences(s)
	if len(sents) > 0 {
		s = sents[0]
	}
	words := strings.Fields(s)
	if len(words) > maxWords {
		s = strings.Join(words[:maxWords], " ") + "."
	}
	return s
}

func formatMMSS(sec int) string {
	if sec < 0 {
		sec = 0
	}
	return fmt.Sprintf("%d:%02d", sec/60, sec%60)
}

func parseMMSS(ts string) int {
	parts := strings.Split(strings.TrimSpace(ts), ":")
	if len(parts) != 2 {
		return 0
	}
	m, _ := strconv.Atoi(parts[0])
	s, _ := strconv.Atoi(parts[1])
	return m*60 + s
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// RegenerateScriptBlock refines a single shot or paragraph block in a realigned documentary script
// strictly via Gemini AI (no local fallback — returns an error if AI is unavailable or fails).
func RegenerateScriptBlock(
	personName string,
	category string,
	biography string,
	summary string,
	fullText string,
	blockIndex int,
	tone string,
	geminiRefiner func(personName, field, existingBlock, biography, tone string) (string, string, error),
) (string, error) {
	if geminiRefiner == nil {
		return "", fmt.Errorf("Gemini AI required for shot refinement: no AI refiner configured")
	}
	blocks := strings.Split(fullText, "\n\n")
	if blockIndex < 0 || blockIndex >= len(blocks) {
		return "", fmt.Errorf("invalid script block index %d", blockIndex)
	}
	target := strings.TrimSpace(blocks[blockIndex])
	if target == "" || target == "---" {
		return "", fmt.Errorf("selected block #%d is empty", blockIndex+1)
	}

	domain := ResolveBiographicalDomain(category, biography, summary, "", "auto")
	lines := strings.Split(target, "\n")

	v, n, err := geminiRefiner(personName, domain.Label, target, biography, tone)
	if err != nil {
		return "", fmt.Errorf("Gemini AI shot refinement failed: %s", sanitizeAIErrorForUI(err))
	}
	v = strings.TrimSpace(v)
	n = strings.TrimSpace(n)
	if v == "" && n == "" {
		return "", fmt.Errorf("Gemini AI returned an empty refinement for block #%d", blockIndex+1)
	}

	// Structured [M:SS] VISUAL ... + NARRATOR (V.O.) shot block
	if len(lines) >= 1 {
		if m := reShotHeader.FindStringSubmatch(strings.TrimSpace(lines[0])); len(m) == 3 {
			ts := m[1]
			newVisual := strings.TrimSpace(m[2])
			if v != "" {
				newVisual = v
			}
			newNarr := ""
			if len(lines) >= 2 && strings.HasPrefix(strings.TrimSpace(lines[1]), "NARRATOR (V.O.):") {
				newNarr = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[1]), "NARRATOR (V.O.):"))
			}
			if n != "" {
				chunks := splitNarrationToSevenSecondChunks(n, MaxWordsPerShot)
				if len(chunks) > 0 {
					newNarr = chunks[0]
				}
			}

			if newNarr != "" {
				blocks[blockIndex] = fmt.Sprintf("[%s] VISUAL: %s  \nNARRATOR (V.O.): %s", ts, newVisual, newNarr)
			} else if len(lines) >= 2 {
				blocks[blockIndex] = fmt.Sprintf("[%s] VISUAL: %s  \n%s", ts, newVisual, strings.TrimSpace(lines[1]))
			}
			return strings.Join(blocks, "\n\n"), nil
		}
	}

	if n == "" {
		return "", fmt.Errorf("Gemini AI returned empty narration for block #%d", blockIndex+1)
	}
	blocks[blockIndex] = n
	return strings.Join(blocks, "\n\n"), nil
}


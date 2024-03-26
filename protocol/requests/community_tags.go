package requests

var tags []string
var tagsEmojis map[string]string
var tagsIndices map[string]uint32

func init() {
	tags = make([]string, 0, len(allTags))
	tagsEmojis = make(map[string]string, len(allTags))
	tagsIndices = make(map[string]uint32, len(allTags))

	for i, pair := range allTags {
		tags = append(tags, pair[0])
		tagsEmojis[pair[0]] = pair[1]
		tagsIndices[pair[0]] = uint32(i)
	}
}

func ValidateTags(input []string) bool {
	for _, t := range input {
		_, ok := tagsEmojis[t]
		if !ok {
			return false
		}
	}

	// False if contains duplicates. Shouldn't have happened
	return len(unique(input)) == len(input)
}

func RemoveUnknownAndDeduplicateTags(input []string) []string {
	var result []string
	for _, t := range input {
		_, ok := tagsEmojis[t]
		if ok {
			result = append(result, t)
		}
	}
	return unique(result)
}

func unique(slice []string) []string {
	uniqMap := make(map[string]struct{})
	for _, v := range slice {
		uniqMap[v] = struct{}{}
	}
	uniqSlice := make([]string, 0, len(uniqMap))
	for v := range uniqMap {
		uniqSlice = append(uniqSlice, v)
	}
	return uniqSlice
}

func TagByIndex(i uint32) string {
	return tags[i]
}

func TagEmoji(tag string) string {
	return tagsEmojis[tag]
}

func TagIndex(tag string) uint32 {
	return tagsIndices[tag]
}

func AvailableTagsCount() int {
	return len(tags)
}

func AvailableTagsEmojis() map[string]string {
	// Make a deep copy of the map to keep it immutable
	emojis := make(map[string]string, len(tagsEmojis))
	for t, e := range tagsEmojis {
		emojis[t] = e
	}
	return emojis
}

var allTags = [][]string{
	{"Activism", "✊"},
	{"Art", "🎨"},
	{"Blockchain", "🔗"},
	{"Books & blogs", "📚"},
	{"Career", "💼"},
	{"Collaboration", "🤝"},
	{"Commerce", "🛒"},
	{"Culture", "🎎"},
	{"DAO", "🚀"},
	{"DeFi", "📈"},
	{"Design", "🧩"},
	{"DIY", "🔨"},
	{"Environment", "🌿"},
	{"Education", "🎒"},
	{"Entertainment", "🍿"},
	{"Ethereum", "Ξ"},
	{"Event", "🗓"},
	{"Fantasy", "🧙‍♂️"},
	{"Fashion", "🧦"},
	{"Food", "🌶"},
	{"Gaming", "🎮"},
	{"Global", "🌍"},
	{"Health", "🧠"},
	{"Hobby", "📐"},
	{"Innovation", "🧪"},
	{"Language", "📜"},
	{"Lifestyle", "✨"},
	{"Local", "📍"},
	{"Love", "❤️"},
	{"Markets", "💎"},
	{"Movies & TV", "🎞"},
	{"Music", "🎶"},
	{"News", "🗞"},
	{"NFT", "🖼"},
	{"Non-profit", "🙏"},
	{"NSFW", "🍆"},
	{"Org", "🏢"},
	{"Pets", "🐶"},
	{"Play", "🎲"},
	{"Podcast", "🎙️"},
	{"Politics", "🗳️"},
	{"Product", "🍱"},
	{"Psyche", "🍁"},
	{"Privacy", "👻"},
	{"Security", "🔒"},
	{"Social", "☕"},
	{"Software dev", "👩‍💻"},
	{"Sports", "⚽️"},
	{"Tech", "📱"},
	{"Travel", "🗺"},
	{"Vehicles", "🚕"},
	{"Web3", "🌐"},
}

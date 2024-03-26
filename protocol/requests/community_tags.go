package requests

var Tags []string
var TagsEmojis map[string]string
var TagsIndices map[string]uint32

func init() {
	orderedInput := [][]string{
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

	Tags = make([]string, 0, len(orderedInput))
	TagsEmojis = make(map[string]string, len(orderedInput))
	TagsIndices = make(map[string]uint32, len(orderedInput))

	for i, pair := range orderedInput {
		Tags = append(Tags, pair[0])
		TagsEmojis[pair[0]] = pair[1]
		TagsIndices[pair[0]] = uint32(i)
	}
}

func ValidateTags(input []string) bool {
	for _, t := range input {
		_, ok := TagsEmojis[t]
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
		_, ok := TagsEmojis[t]
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

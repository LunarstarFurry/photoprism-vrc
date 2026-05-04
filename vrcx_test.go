package main

import (
	"fmt"
	"regexp"
)

var (
	vrcxWorldRegexp      = regexp.MustCompile(`"?world"?\s*:\s*\{[^}]*"?name"?\s*:\s*"?([^,"}]+)"?`)
	vrcxAuthorRegexp     = regexp.MustCompile(`"?author"?\s*:\s*\{[^}]*"?displayName"?\s*:\s*"?([^,"}]+)"?`)
	vrcxPlayersRegexp    = regexp.MustCompile(`"?players"?\s*:\s*\[(.*?)\]`)
	vrcxPlayerNameRegexp = regexp.MustCompile(`"?displayName"?\s*:\s*"?([^,"}]+)"?`)
)

func main() {
	s := `{application:VRCX,version:1,author:{id:usr_b124255b-f0de-441d-a019-5e8a690c7d29,displayName:LunarstarFurry},world:{name:SlashCo VR,id:wrld_41efe3b1-9931-40ab-a15d-6946d22481b5,instanceId:wrld_41efe3b1-9931-40ab-a15d-6946d22481b5:06017~group(grp_63f20923-a536-4330-a137-dadf2ebac45a)~groupAccessType(public)~region(use)},players:[{id:usr_ebc2f9c5-ae53-42dc-865c-7d3eecc3af8e,displayName:BuppyBabs},{id:usr_31ba26cc-366c-4f70-bc06-7744183d7220,displayName:~Scarlet Viper~},{id:usr_1addbda5-e769-41d2-aab4-0b94c3f2489e,displayName:dimka50055},{id:usr_c81b1f39-f7fe-4646-a445-6c3ddd7bfea4,displayName:Experimentfluff},{id:usr_b124255b-f0de-441d-a019-5e8a690c7d29,displayName:LunarstarFurry},{id:usr_cb6a1e76-4d55-49fb-b8d3-a0a1e05cca36,displayName:kuniafterlife},{id:usr_58e75e3a-98ea-4057-933d-b8bff7d438ca,displayName:VibeCheckTM}]}`
	
	m := vrcxWorldRegexp.FindStringSubmatch(s)
	fmt.Println("World:", len(m))

	m = vrcxAuthorRegexp.FindStringSubmatch(s)
	fmt.Println("Author:", len(m))

	m = vrcxPlayersRegexp.FindStringSubmatch(s)
	fmt.Println("Players:", len(m))
}

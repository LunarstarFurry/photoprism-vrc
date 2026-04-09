package meta

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeUnicode(t *testing.T) {
	t.Run("Ascii", func(t *testing.T) {
		assert.Equal(t, "IMG_0599", SanitizeUnicode("IMG_0599"))
	})
	t.Run("Unicode", func(t *testing.T) {
		assert.Equal(t, "Naïve bonds and futures surge as inflation eases 🚀🚀🚀", SanitizeUnicode("  Naïve bonds and futures surge as inflation eases 🚀🚀🚀 "))
	})
	t.Run("Empty", func(t *testing.T) {
		assert.Equal(t, "", SanitizeUnicode(""))
	})
}

func TestSanitizeTitle(t *testing.T) {
	t.Run("ImgNum0599", func(t *testing.T) {
		result := SanitizeTitle("IMG_0599")

		if result != "" {
			t.Fatal("result should be empty")
		}
	})
	t.Run("ImgNum0599Jpg", func(t *testing.T) {
		result := SanitizeTitle("IMG_0599.JPG")

		if result != "" {
			t.Fatal("result should be empty")
		}
	})
	t.Run("ImgNum0599Abc", func(t *testing.T) {
		result := SanitizeTitle("IMG_0599 ABC")

		if result != "IMG_0599 ABC" {
			t.Fatal("result should be IMG_0599 ABC")
		}
	})
	t.Run("DSC10599", func(t *testing.T) {
		result := SanitizeTitle("DSC10599")

		if result != "" {
			t.Fatal("result should be empty")
		}
	})
	t.Run("TitanicCloudComputingJpg", func(t *testing.T) {
		result := SanitizeTitle("titanic_cloud_computing.jpg")

		assert.Equal(t, "Titanic Cloud Computing", result)
	})
	t.Run("NaomiWattsEwanMcgregorTheImpossibleTiffNum2012Num7999540939OJpg", func(t *testing.T) {
		result := SanitizeTitle("naomi-watts--ewan-mcgregor--the-impossible--tiff-2012_7999540939_o.jpg")

		assert.Equal(t, "Naomi Watts / Ewan McGregor / The Impossible / TIFF", result)
	})
	t.Run("BeiDenLandungsbrCkenPng", func(t *testing.T) {
		result := SanitizeTitle("Bei den Landungsbrücken.png")

		assert.Equal(t, "Bei den Landungsbrücken", result)
	})
	t.Run("BeiDenLandungsbrCkenFoo", func(t *testing.T) {
		result := SanitizeTitle("Bei den Landungsbrücken.foo")

		assert.Equal(t, "Bei den Landungsbrücken.foo", result)
	})
	t.Run("LetItSnow", func(t *testing.T) {
		result := SanitizeTitle("let_it_snow")

		assert.Equal(t, "let_it_snow", result)
	})
	t.Run("LetItSnowJpg", func(t *testing.T) {
		result := SanitizeTitle("let_it_snow.jpg")

		assert.Equal(t, "Let It Snow", result)
	})
	t.Run("NiklausWirthJpg", func(t *testing.T) {
		result := SanitizeTitle("Niklaus_Wirth.jpg")

		assert.Equal(t, "Niklaus Wirth", result)
	})
	t.Run("NiklausWirth", func(t *testing.T) {
		result := SanitizeTitle("Niklaus_Wirth")

		assert.Equal(t, "Niklaus_Wirth", result)
	})
	t.Run("StringWithBinaryData", func(t *testing.T) {
		result := SanitizeTitle("string with binary data blablabla")

		assert.Equal(t, "", result)
	})
}

func TestSanitizeCaption(t *testing.T) {
	t.Run("VRCX_Formatting", func(t *testing.T) {
		raw := `{application:VRCX,version:1,author:{id:usr_b124255b-f0de-441d-a019-5e8a690c7d29,displayName:LunarstarFurry},world:{name:SlashCo VR,id:wrld_41efe3b1-9931-40ab-a15d-6946d22481b5,instanceId:wrld_41efe3b1-9931-40ab-a15d-6946d22481b5:54309~group(grp_63f20923-a536-4330-a137-dadf2ebac45a)~groupAccessType(public)~region(use)},players:[{id:usr_7deb0523-fec8-452a-bb13-413b02ed4337,displayName:Delboy},{id:usr_4dbe9c53-3c1e-4b8d-974b-e50da280c628,displayName:Riletin},{id:usr_b124255b-f0de-441d-a019-5e8a690c7d29,displayName:LunarstarFurry},{id:usr_41c22689-82ce-4075-9cf5-a38805803b83,displayName:Ivy},{id:usr_6fdd431f-def0-40ad-a28d-feba46df123e,displayName:ThatWinterBoi},{id:usr_c04da893-7953-4ddf-a29c-abe0cf3b0833,displayName:FishbonesX2},{id:usr_54f7d04c-b7ec-4902-a7f3-e6b9eabf8820,displayName:Tengoku Hoshi},{id:usr_55ff07e1-4545-4ea3-a9a1-2c4365e1441a,displayName:SolaceSeen}]}`
		expected := "VRChat - World: SlashCo VR | Author: LunarstarFurry | Players: Delboy, Riletin, LunarstarFurry, Ivy, ThatWinterBoi, FishbonesX2, Tengoku Hoshi, SolaceSeen"
		result := SanitizeCaption(raw)
		assert.Equal(t, expected, result)
	})
	t.Run("VRCX_Formatting_Quotes", func(t *testing.T) {
		raw := `{"application":"VRCX","version":1,"author":{"id":"usr_123","displayName":"LunarstarFurry"},"world":{"name":"SlashCo VR","id":"wrld_123"},"players":[{"id":"usr_456","displayName":"Delboy"}]}`
		expected := "VRChat - World: SlashCo VR | Author: LunarstarFurry | Players: Delboy"
		result := SanitizeCaption(raw)
		assert.Equal(t, expected, result)
	})
	t.Run("ImgNum0599", func(t *testing.T) {
		result := SanitizeCaption("IMG_0599")

		if result == "" {
			t.Fatal("result should not be empty")
		}
	})
	t.Run("OlympusDigitalCamera", func(t *testing.T) {
		result := SanitizeCaption("OLYMPUS DIGITAL CAMERA")

		if result != "" {
			t.Fatal("result should be empty")
		}
	})
	t.Run("GoPro", func(t *testing.T) {
		result := SanitizeCaption("DCIM\\108GOPRO\\GOPR2137.JPG")

		if result != "" {
			t.Fatal("result should be empty")
		}
	})
	t.Run("Hdrpl", func(t *testing.T) {
		result := SanitizeCaption("hdrpl")

		if result != "" {
			t.Fatal("result should be empty")
		}
	})
	t.Run("Btf", func(t *testing.T) {
		result := SanitizeCaption("btf")

		if result != "" {
			t.Fatal("result should be empty")
		}
	})
	t.Run("Wtf", func(t *testing.T) {
		result := SanitizeCaption("wtf")

		if result != "wtf" {
			t.Fatal("result should be 'wtf'")
		}
	})
}

func TestSanitizeUID(t *testing.T) {
	t.Run("Num77D9a719ede3f95915abd081d7b7cb2c", func(t *testing.T) {
		result := SanitizeUID("77d9a719ede3f95915abd081d7b7CB2c")
		assert.Equal(t, "77d9a719ede3f95915abd081d7b7cb2c", result)
	})
	t.Run("Num77D", func(t *testing.T) {
		result := SanitizeUID("77d")
		assert.Equal(t, "", result)
	})
	t.Run("Num77D9a719ede3f95915abd081d7b7cb2c", func(t *testing.T) {
		result := SanitizeUID(":77d9a719ede3f95915abd081d7b7CB2c")
		assert.Equal(t, "77d9a719ede3f95915abd081d7b7cb2c", result)
	})

}

import Testing

@testable import StarterPackApp

struct StarterPackTests {
    @Test
    func starterPackNameIsAvailable() {
        #expect(starterPackName == "StarterPack")
    }
}

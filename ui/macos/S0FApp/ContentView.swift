import SwiftUI

struct ContentView: View {
    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("s0f")
                .font(.largeTitle)
            Text("SwiftUI shell coming soon.")
                .foregroundColor(.secondary)
            Text("This target will speak IPC to the Go daemon once the transport is ready.")
                .font(.footnote)
        }
        .padding()
        .frame(minWidth: 480, minHeight: 320)
    }
}

#Preview {
    ContentView()
}

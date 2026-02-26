class Dnsimple < Formula
  desc "Terminal-first DNSimple client (TUI + CLI)"
  homepage "https://github.com/dorkitude/simple"
  license "MIT"
  head "https://github.com/dorkitude/simple.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(output: bin/"dnsimple"), "."
    bin.install_symlink "dnsimple" => "simple"
    bin.install_symlink "dnsimple" => "dnsimplectl"
  end

  test do
    assert_match "DNSimple", shell_output("#{bin}/dnsimple --help")
  end
end

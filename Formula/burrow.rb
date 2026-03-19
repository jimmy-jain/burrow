class Burrow < Formula
  desc "macOS System Maintenance CLI"
  homepage "https://github.com/jimmy-jain/burrow"
  url "https://github.com/jimmy-jain/burrow/archive/refs/tags/V0.1.0.tar.gz"
  sha256 "PLACEHOLDER"
  license "MIT"
  head "https://github.com/jimmy-jain/burrow.git", branch: "main"

  depends_on :macos
  depends_on "go" => :build

  def install
    system "make", "build"

    # Main CLI + alias
    bin.install "burrow"
    bin.install "bw"

    # Go binaries
    bin.install "bin/analyze-go" => "burrow-analyze"
    bin.install "bin/status-go" => "burrow-status"
    bin.install "bin/dupes-go" => "burrow-dupes"
    bin.install "bin/watch-go" => "burrow-watch"
    bin.install "bin/burrow-mcp"

    # Shell libraries (used by burrow at runtime)
    libexec.install Dir["bin/*.sh"]
    libexec.install "lib"

    # Patch SCRIPT_DIR fallback so burrow finds libs under libexec
    inreplace bin/"burrow", /^SCRIPT_DIR=.*$/, "SCRIPT_DIR=\"#{libexec}\""
  end

  def caveats
    <<~EOS
      To enable shell tab completion:
        burrow completion

      To enable shell cd-hook integration:
        eval "$(burrow hook bash)"   # or zsh/fish

      To set up scheduled maintenance:
        burrow schedule install

      If you previously used Mole, Burrow will auto-migrate
      your config from ~/.config/mole on first run.
    EOS
  end

  test do
    assert_match "Burrow version", shell_output("#{bin}/burrow --version")
    assert_match "burrow", shell_output("#{bin}/bw --version")
  end
end

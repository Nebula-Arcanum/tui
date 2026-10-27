class Zn < Formula
  desc "ZenNotes command-line interface and terminal app for Markdown notes"
  homepage "https://github.com/ZenNotes/tui"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ZenNotes/tui/releases/download/v0.12.1/zn_0.12.1_darwin_arm64.tar.gz"
      sha256 "ab931a3ee5acc91bcd6214cb51307563c99f230e171631a49b51ce4ad0a5b9a3"
    end
    on_intel do
      url "https://github.com/ZenNotes/tui/releases/download/v0.12.1/zn_0.12.1_darwin_amd64.tar.gz"
      sha256 "c61358d55dcdcd325560d6cadefa1e9404e8f1f43ebe78905bb79f77040fcebe"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ZenNotes/tui/releases/download/v0.12.1/zn_0.12.1_linux_arm64.tar.gz"
      sha256 "956bfc2b6a4c3b5feac55b1a0ac54537c2af2fddac0502db1ad899bb1c7edc49"
    end
    on_intel do
      url "https://github.com/ZenNotes/tui/releases/download/v0.12.1/zn_0.12.1_linux_amd64.tar.gz"
      sha256 "cbb2b832bc0ae15c7d0b2392346dfa41e918c9974d628e0cb40ff88577687f5a"
    end
  end

  def install
    bin.install "zn"
  end

  def caveats
    <<~EOS
      zn renders display math with typst by default. To draw LaTeX math instead
      (math = "latex" under [terminal] in config.toml), zn runs pdftex and
      pdftocairo installed on your computer; it does not include TeX. On macOS:
        brew install --cask basictex
        brew install poppler
        sudo tlmgr install standalone varwidth preview xcolor amsmath amsfonts mhchem pgf
      On Linux, install your distribution's TeX Live and poppler packages.
      Run zn doctor to check; it prints the commands for your system.
    EOS
  end

  test do
    ENV["ZENNOTES_CONFIG_DIR"] = (testpath/"config").to_s
    assert_match version.to_s, shell_output("#{bin}/zn --version")

    system bin/"zn", "init", testpath/"notes"
    (testpath/"notes/Homebrew.md").write("# Homebrew\n\nA note from the formula test.\n")
    assert_equal "# Homebrew\n\nA note from the formula test.\n",
                 shell_output("#{bin}/zn read Homebrew.md --vault #{testpath}/notes")
  end
end

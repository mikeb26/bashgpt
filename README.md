# BashGPT
Bash shell auto-completion via OpenAI's GPT API

## Building

```bash
make
```

## Installing

```bash
mkdir -p $HOME/bin
BASHGPT=$(curl -s https://api.github.com/repos/mikeb26/bashgpt/releases/latest | grep browser_download_url | cut -f4 -d\")
wget $BASHGPT
chmod 755 bashgpt
mv bashgpt $HOME/bin
# add $HOME/bin to your $PATH if not already present
bashgpt config
```

## Usage

```bash
bashgpt - Bash shell auto-completion via OpenAI's GPT API

Usage:
  ? <PROMPT>
  bashgpt <COMMAND>

Available Commands:
  config                         Initial setup of bashgpt (set API key)
  help                           This help screen
  upgrade                        Upgrade to the latest version of bashgpt
  sh <PROMPT> [-- <RESPONSE>]    Generate a bash script from PROMPT. '? <PROMPT>'
                                 is a shortcut for 'bashgpt sh <PROMPT>'
  version                        Print the current version of bashgpt and exit

PROMPT:
  The goal you are trying to accomplish via the shell

RESPONSE:
  The previously generated response from a PROMPT. When present, bashgpt will ignore the PROMPT
  and instead just execute RESPONSE. This is done in order to work within the limitations of
  bash programmable completion.

EXAMPLES:

  bash$ ? rotate foo.png 90 degrees, convert to greyscale, and reencode as jpeg 
  convert -rotate 90 -type grayscale foo.png foo.jpg

  bash$ ? extract a 34s audio clip from bar.mpeg at offset 1m43s and encode in FLAC
  ffmpeg -ss 00:01:43 -i bar.mpeg -t 00:00:34 -c:a flac output.flac

  bash$ ? find all files under /usr larger than 128KiB, were last modified between aug 2022 and mar 2023, and are not owned by root
  find /usr -type f -size +128k ! -user root -newermt "Aug 1, 2022" ! -newermt "Mar 31, 2023"

  bash$ ? download ubuntu 20.04 iso
  wget --user-agent="Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4183.102 Safari/537.36" https://releases.ubuntu.com/20.04.1/ubuntu-20.04.1-desktop-amd64.iso
```

## Contributing
Pull requests are welcome at https://github.com/mikeb26/bashgpt

For major changes, please open an issue first to discuss what you
would like to change.

## License
[AGPL3](https://www.gnu.org/licenses/agpl-3.0.en.html)

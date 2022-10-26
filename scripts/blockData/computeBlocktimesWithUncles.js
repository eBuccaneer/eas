const fs = require('fs');
const path = require('path');
const readline = require('readline');

const delay = ms => new Promise(resolve => setTimeout(resolve, ms))

async function main() {
  const blockTimestampWithUnclesPath = path.join(__dirname, '.', 'out', 'blocktimestamps_withUncles.txt');
  const blockTimestampWithUnclesStream = fs.createReadStream(blockTimestampWithUnclesPath);
  const blockTimePath = path.join(__dirname, '.', 'out', 'blocktimes_withUncles.txt');
  const blockTimeStream = fs.createWriteStream(blockTimePath, {flags:'w'});
  const blockTimestampWithUncles = []

  const rl = readline.createInterface({
    input: blockTimestampWithUnclesStream,
    crlfDelay: Infinity
  });
  // Note: we use the crlfDelay option to recognize all instances of CR LF
  // ('\r\n') in input.txt as a single line break.

  for await (const line of rl) {
    blockTimestampWithUncles.push(line)
  }


  blockTimestampWithUncles.sort();
  let lastSec = 0;
  for (let i = 0; i < blockTimestampWithUncles.length; i++) {
    if (lastSec !== 0) {
      blockTimeStream.write(`${blockTimestampWithUncles[i] - lastSec}\n`);
    }
    lastSec = blockTimestampWithUncles[i]
  }
}

main()
  .catch(err => console.error(err));

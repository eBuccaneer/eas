const Web3 = require("web3");
const fs = require('fs');
const path = require('path');
const ethNetwork = "INFURA_ENDPOINT"; // set infura endpoint here
const web3 = new Web3(new Web3.providers.HttpProvider(ethNetwork));
const lastDaysToCover = 1; // edit days to query here
const blocksPerDay = 1000;
let blocksToQueryAtOnce = 1000;
const blockAmount = lastDaysToCover * blocksPerDay;

const delay = ms => new Promise(resolve => setTimeout(resolve, ms))

async function main() {
  const latestBlock = await web3.eth.getBlockNumber();
  let requestsRemaining = blockAmount;
  const miners = new Map();
  const blockTimes = [];
  const blockTimesWithUncles = [];
  const uncleNumbersPerQueryPeriod = [];
  const uncleHashes = [];
  const blocksWithUncles = [];
  let txHashes = [];
  let errorCount = 0;
  const logPath = path.join(__dirname, '.', 'out', 'log.txt');
  const logStream = fs.createWriteStream(logPath, {flags:'w'});
  const blockTimePath = path.join(__dirname, '.', 'out', 'blocktimes.txt');
  const blockTimeStream = fs.createWriteStream(blockTimePath, {flags:'w'});
  const blockTimestampPath = path.join(__dirname, '.', 'out', 'blocktimestamps.txt');
  const blockTimestampStream = fs.createWriteStream(blockTimestampPath, {flags:'w'});
  const blockTimestampWithUnclesPath = path.join(__dirname, '.', 'out', 'blocktimestamps_withUncles.txt');
  const blockTimestampWithUnclesStream = fs.createWriteStream(blockTimestampWithUnclesPath, {flags:'w'});
  const unclesPerDayPath = path.join(__dirname, '.', 'out', 'unclesPerDay.txt');
  const unclesPerDayStream = fs.createWriteStream(unclesPerDayPath, {flags:'w'});
  const uncleHashesPath = path.join(__dirname, '.', 'out', 'uncleHashes.txt');
  const uncleHashesStream = fs.createWriteStream(uncleHashesPath, {flags:'w'});
  const txGasPath = path.join(__dirname, '.', 'out', 'txGas.txt');
  const txGasStream = fs.createWriteStream(txGasPath, {flags:'w'});
  const txGasPricePath = path.join(__dirname, '.', 'out', 'txGasPrice.txt');
  const txGasPriceStream = fs.createWriteStream(txGasPricePath, {flags:'w'});
  const errorsPath = path.join(__dirname, '.', 'out', 'errors.txt');
  const errorsStream = fs.createWriteStream(errorsPath, {flags:'w'});
  process.stdout.write(`waiting for ${requestsRemaining} requests to complete...\n`)
  process.stdout.write(`remaining requests\t\t\t\tfound miners\t\tblock response errors\n`);
  for (let i = latestBlock; i > latestBlock - blockAmount; i -= blocksToQueryAtOnce){
    process.stdout.write(`${requestsRemaining}\t\t\t\t\t\t${miners.size}\t\t\t${errorCount}`);
    const amountToQuery = Math.min(blocksToQueryAtOnce, requestsRemaining)
    errorCount = await processNStartingWith(amountToQuery, i, miners, blockTimes, blockTimesWithUncles, uncleNumbersPerQueryPeriod, uncleHashes, errorCount, errorsStream, uncleHashesStream, blockTimestampStream, blockTimestampWithUnclesStream, blocksWithUncles, txHashes, txGasStream, txGasPriceStream);

    let txRequestsRemaining = txHashes.length;
    for (let i = 0; i < txHashes.length; i += blocksToQueryAtOnce){
      const amountToQuery = Math.min(blocksToQueryAtOnce, txRequestsRemaining)
      await processNTransactionGasPrice(amountToQuery, i, errorCount, errorsStream, txHashes, txGasPriceStream);
      await processNTransactionGas(amountToQuery, i, errorCount, errorsStream, txHashes, txGasStream);
    }
    txHashes = [];
    requestsRemaining -= amountToQuery;
    process.stdout.clearLine();
    process.stdout.cursorTo(0);
  }
  process.stdout.write(`0\t\t\t\t\t\t${miners.size}\t\t\t${errorCount}`);
  process.stdout.write("\n")

  let uncleRequestsRemaining = uncleHashes.length;
  blocksToQueryAtOnce = blocksToQueryAtOnce / 2;
  process.stdout.write(`waiting for ${uncleRequestsRemaining} uncle requests to complete...\n`)
  for (let i = 0; i < uncleHashes.length; i += blocksToQueryAtOnce){
    process.stdout.write(`${uncleRequestsRemaining}\t\t\t\t\t\t${miners.size}\t\t\t${errorCount}`);
    const amountToQuery = Math.min(blocksToQueryAtOnce, uncleRequestsRemaining)
    errorCount = await processNUnclesStartingWith(amountToQuery, i, miners, blockTimesWithUncles, uncleHashes, errorCount, errorsStream, blockTimestampWithUnclesStream, blocksWithUncles);
    uncleRequestsRemaining -= amountToQuery;
    process.stdout.clearLine();
    process.stdout.cursorTo(0);
  }

  process.stdout.write(`0\t\t\t\t\t\t${miners.size}\t\t\t${errorCount}`);
  process.stdout.write("\n")
  blockTimes.sort();
  let lastSec = 0;
  for (let i = 0; i < blockTimes.length; i++) {
    if (lastSec !== 0) {
      blockTimeStream.write(`${blockTimes[i] - lastSec}\n`);
    }
    lastSec = blockTimes[i]
  }
  for (let i = 0; i < uncleNumbersPerQueryPeriod.length; i++) {
    unclesPerDayStream.write(`${uncleNumbersPerQueryPeriod[i]}\n`)
  }
  console.log(`Found ${miners.size} different miners in last ${blockAmount} blocks`);
  logStream.write(`Found ${miners.size} different miners in last ${blockAmount} blocks\n`);
  const minersArraySorted = []
  for (const [key, value] of miners) {
    minersArraySorted.push({address: key, blocksMined: value})
  }
  minersArraySorted.sort(function(a, b) {
    return b.blocksMined - a.blocksMined;
  });
  for (const stat of minersArraySorted) {
    console.log(`${stat.address}: ${stat.blocksMined} / ${(stat.blocksMined / blockAmount * 100).toFixed(5)} %`)
    logStream.write(`${stat.address}: ${stat.blocksMined} / ${(stat.blocksMined / blockAmount * 100).toFixed(5)} %\n`);
  }
}

async function processNStartingWith(n, blockNumberToStart, minersMap, blockTimes, blockTimesWithUncles, uncleNumbers, uncleHashes, errorCount, errorsStream, uncleHashesStream, blockTimestampStream, blockTimestampWithUnclesStream, blocksWithUncles, txHashes) {
  const queryPeriodDaysFactor = blocksPerDay / n;
  const batch = new web3.eth.BatchRequest();
  let requestsRemaining = n;
  let unclesThisPeriod = 0;
  let missingBlocks = [];
  for (let i = blockNumberToStart; i > blockNumberToStart - n; i--) {
    batch.add(web3.eth.getBlock.request(i, (err, res) => {
      const b = i;
      if (res) {
        if(res.timestamp) {
          blockTimes.push(res.timestamp)
          blockTimesWithUncles.push(res.timestamp)
          blockTimestampStream.write(`${res.timestamp}\n`);
          blockTimestampWithUnclesStream.write(`${res.timestamp}\n`);
        }
        if(res.uncles && res.uncles.length > 0) {
          blocksWithUncles.push({blockNumber: b, amount: res.uncles.length});
          unclesThisPeriod += res.uncles.length
          for(let i = 0; i < res.uncles.length; i++) {
            uncleHashes.push(res.uncles[i])
            uncleHashesStream.write(`${res.uncles[i]}\n`);
          }
        }
        if (minersMap.has(res.miner)) {
          minersMap.set(res.miner, minersMap.get(res.miner) + 1);
        } else {
          minersMap.set(res.miner, 1);
        }
        if (res.transactions.length > 0) {
          txHashes.push(...res.transactions)
        }
      } else {
        errorCount++;
        if (err) {
          errorsStream.write(`${err.toString()}\n`);
        }
        missingBlocks.push(b);
        errorsStream.write(`block missing: ${b}\n`);
      }
      requestsRemaining --;
    }))
  }
  await batch.execute();
  while (true) {
    await delay(1000);
    if (requestsRemaining === 0) {
      if (missingBlocks.length > 0) {
        const batch = new web3.eth.BatchRequest();
        requestsRemaining = missingBlocks.length;
        for (let j = 0; j < missingBlocks.length; j++) {

          batch.add(web3.eth.getBlock.request(missingBlocks[j], (err, res) => {
            const b = missingBlocks[j];
            if (res) {
              if(res.timestamp) {
                blockTimes.push(res.timestamp)
                blockTimesWithUncles.push(res.timestamp)
                blockTimestampStream.write(`${res.timestamp}\n`);
                blockTimestampWithUnclesStream.write(`${res.timestamp}\n`);
              }
              if(res.uncles && res.uncles.length > 0) {
                blocksWithUncles.push({blockNumber: b, amount: res.uncles.length});
                unclesThisPeriod += res.uncles.length
                for(let i = 0; i < res.uncles.length; i++) {
                  uncleHashes.push(res.uncles[i])
                  uncleHashesStream.write(`${res.uncles[i]}\n`);
                }
              }
              if (minersMap.has(res.miner)) {
                minersMap.set(res.miner, minersMap.get(res.miner) + 1);
              } else {
                minersMap.set(res.miner, 1);
              }
            } else {
              errorCount++;
              if (err) {
                errorsStream.write(`${err.toString()}\n`);
              }
              missingBlocks.push(b);
              errorsStream.write(`block missing again: ${b}\n`);
            }
            requestsRemaining --;
          }))
        }
        await batch.execute();
        while (true) {
          await delay(1000);
          if (requestsRemaining === 0) {
            break;
          }
        }
      }

      uncleNumbers.push(unclesThisPeriod * queryPeriodDaysFactor)
      break;
    }
  }
  return Promise.resolve(errorCount);
}

async function processNTransactionGasPrice(n, indexToStart, errorCount, errorsStream, txHashes, txGasPriceStream) {
  const batch = new web3.eth.BatchRequest();
  let requestsRemaining = 0;
  for (let i = indexToStart; i < indexToStart + n; i++) {
    if (txHashes[i]) {
      requestsRemaining ++;
      batch.add(web3.eth.getTransaction.request(txHashes[i], (err, res) => {
        const t = txHashes[i];
        if (res) {
          txGasPriceStream.write(`${res.gasPrice}\n`)
        } else {
          errorCount++;
          if (err) {
            errorsStream.write(`${err.toString()}\n`);
          }
          errorsStream.write(`tx missing: ${t}\n`);
        }
        requestsRemaining--;
      }))
    }
  }
  await batch.execute();
  while (true) {
    await delay(1000);
    if (requestsRemaining === 0) {
      break;
    }
  }
  return Promise.resolve(errorCount);
}

async function processNTransactionGas(n, indexToStart, errorCount, errorsStream, txHashes, txGasStream) {
  const batch = new web3.eth.BatchRequest();
  let requestsRemaining = 0;
  for (let i = indexToStart; i < indexToStart + n; i++) {
    if (txHashes[i]) {
      requestsRemaining ++;
      batch.add(web3.eth.getTransactionReceipt.request(txHashes[i], (err, res) => {
        const t = txHashes[i];
        if (res) {
          txGasStream.write(`${res.gasUsed}\n`)
        } else {
          errorCount++;
          if (err) {
            errorsStream.write(`${err.toString()}\n`);
          }
          errorsStream.write(`tx missing: ${t}\n`);
        }
        requestsRemaining--;
      }))
    }
  }
  await batch.execute();
  while (true) {
    await delay(1000);
    if (requestsRemaining === 0) {
      break;
    }
  }
  return Promise.resolve(errorCount);
}

async function processNUnclesStartingWith(n, indexToStart, minersMap, blockTimes, uncleHashes, errorCount, errorsStream, blockTimestampStream, blocksWithUncles) {
  const batch = new web3.eth.BatchRequest();
  let requestsRemaining = 0;
  let missingUncles = [];
  for (let i = indexToStart; i < indexToStart + n; i++) {
    if (blocksWithUncles[i]) {
      for (let j = 0; j < blocksWithUncles[i].amount; j++) {
        requestsRemaining ++;
        batch.add(web3.eth.getUncle.request(blocksWithUncles[i].blockNumber, j, (err, res) => {
          if (res) {
            if(res.timestamp) {
              blockTimes.push(res.timestamp)
              blockTimestampStream.write(`${res.timestamp}\n`);
            }
            if (minersMap.has(res.miner)) {
              minersMap.set(res.miner, minersMap.get(res.miner) + 1);
            } else {
              minersMap.set(res.miner, 1);
            }
          } else {
            errorCount++;
            if (err) {
              errorsStream.write(`${err.toString()}\n`);
            }
            missingUncles.push(blocksWithUncles[i]);
            errorsStream.write(`uncle missing: ${blocksWithUncles[i].blockNumber} - ${j}\n`);
          }
          requestsRemaining --;
        }));
      }
    }
  }
  await batch.execute();
  while (true) {
    await delay(1000);
    if (requestsRemaining === 0) {
      if (missingUncles.length > 0) {
        const batch = new web3.eth.BatchRequest();
        requestsRemaining = missingUncles.length;
        for (let i = 0; i < missingUncles.length; i++) {
          if (missingUncles[i]) {
            for (let j = 0; j < missingUncles[i].amount; j++) {
              requestsRemaining ++;
              batch.add(web3.eth.getUncle.request(missingUncles[i].blockNumber, j, (err, res) => {
                if (res) {
                  if(res.timestamp) {
                    blockTimes.push(res.timestamp)
                    blockTimestampStream.write(`${res.timestamp}\n`);
                  }
                  if (minersMap.has(res.miner)) {
                    minersMap.set(res.miner, minersMap.get(res.miner) + 1);
                  } else {
                    minersMap.set(res.miner, 1);
                  }
                } else {
                  errorCount++;
                  if (err) {
                    errorsStream.write(`${err.toString()}\n`);
                  }
                  errorsStream.write(`uncle missing again: ${missingUncles[i].blockNumber} - ${j}\n`);
                }
                requestsRemaining --;
              }));
            }
          }
        }
        await batch.execute();
        while (true) {
          await delay(1000);
          if (requestsRemaining === 0) {
            break;
          }
        }
      }
      break;
    }
  }
  return Promise.resolve(errorCount);
}

main()
  .catch(err => console.error(err));

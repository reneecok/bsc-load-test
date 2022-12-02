const fs = require('fs');
const {expect} = require("chai");

module.exports = {
    writeJsonObject,
    readJsonObject,
    checkTransStatus
};

function writeJsonObject(filepath, jsonObj) {
    fs.writeFileSync(
        filepath,
        JSON.stringify(jsonObj, null, 4) // Indent 4 spaces
    )
}

function readJsonObject(filepath) {
    return JSON.parse(
        fs.readFileSync(filepath).toString()
    )
}

async function checkTransStatus(tx) {
    let receipt = await tx.wait();
    expect(receipt.status).equal(1);
}
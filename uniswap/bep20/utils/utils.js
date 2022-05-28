const fs = require('fs');

module.exports = {
    writeJsonObject,
    readJsonObject
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
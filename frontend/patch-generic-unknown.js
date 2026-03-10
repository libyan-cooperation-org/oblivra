const fs = require('fs');
const path = require('path');

function walk(dir) {
    let results = [];
    const list = fs.readdirSync(dir);
    list.forEach(function (file) {
        file = path.join(dir, file);
        const stat = fs.statSync(file);
        if (stat && stat.isDirectory()) {
            results = results.concat(walk(file));
        } else if (file.endsWith('.tsx') || file.endsWith('.ts')) {
            results.push(file);
        }
    });
    return results;
}

const allFiles = walk(path.join(__dirname, 'src'));

for (const file of allFiles) {
    let content = fs.readFileSync(file, 'utf8');
    let original = content;

    // Catch block Error casts
    content = content.replace(/err\.message/g, "(err as Error).message");
    content = content.replace(/e\.message/g, "(e as Error).message");
    content = content.replace(/err\.toString\(\)/g, "String(err)");
    content = content.replace(/e\.toString\(\)/g, "String(e)");
    content = content.replace(/re\.toString\(\)/g, "String(re)");

    if (content !== original) {
        fs.writeFileSync(file, content);
    }
}
console.log("Patched catch blocks");

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

    // Revert `((e as Error) || {}).currentTarget` back to `e.currentTarget`
    content = content.replace(/\(\(e as Error\) \|\| \{\}\)\./g, "e.");
    content = content.replace(/\(\(err as Error\) \|\| \{\}\)\./g, "err.");

    content = content.replace(/\(\(e as Error\) \|\| \{\}\)\?/g, "e?");
    content = content.replace(/\(\(err as Error\) \|\| \{\}\)\?/g, "err?");

    if (content !== original) {
        fs.writeFileSync(file, content);
    }
}
console.log("Reverted bad error casts");

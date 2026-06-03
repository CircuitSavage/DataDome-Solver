/**
 * DataDome tags.js AST-Based Deobfuscator v2
 *
 * Babel AST transformations to fully decode and clean the obfuscated code.
 *
 * Layers handled:
 *   1. f1[]/s1[] string table extraction + A(n)/J(n) inlining
 *   2. MBA helper resolution (W1, T1, M1, O1, P1, d1, B1, G1, Q1, A1, x1, O, ...)
 *   3. j.Number(x), j.Math.floor(x), j.Math.ceil(x), j.parseInt(x) → constant
 *   4. Constant folding (binary, unary, void 0 → undefined)
 *   5. Bracket → dot notation
 *   6. typeof normalization
 *   7. Dead-code removal (expression statements that are just literals/calls with no effect)
 *
 * Usage: node deobfuscator.js [input] [output]
 */

const fs = require('fs');
const parser = require('@babel/parser');
const traverse = require('@babel/traverse').default;
const generate = require('@babel/generator').default;
const t = require('@babel/types');

const inputFile = process.argv[2] || 'tags.js';
const outputFile = process.argv[3] || 'tags_deobfuscated.js';
const source = fs.readFileSync(inputFile, 'utf-8');

// ═══════════════════════════════════════════════════════════════════════════════
// Decoders
// ═══════════════════════════════════════════════════════════════════════════════

const CUSTOM_ALPHABET = 'H1DAxCvrj7IaPRL8GSJZKX3f62e9d0VTilFEOWgUB=/t+QmMwuskNnhpb4oyq5Yzc';

function decodeJ(idx, f1arr) {
    if (idx < 0 || idx >= f1arr.length) return undefined;
    const val = f1arr[idx];
    if (typeof val !== 'string') return undefined;
    return Buffer.from(val, 'base64').toString('utf-8');
}

function decodeA(idx, s1arr) {
    if (idx < 0 || idx >= s1arr.length) return undefined;
    const val = s1arr[idx];
    if (typeof val !== 'string') return val;
    let result = '', cleaned = val.replace(/[^A-Za-z0-9\+\/\=]/g, ''), pos = 0;
    while (pos < cleaned.length) {
        const o = CUSTOM_ALPHABET.indexOf(cleaned.charAt(pos++));
        const r = CUSTOM_ALPHABET.indexOf(cleaned.charAt(pos++));
        const i = CUSTOM_ALPHABET.indexOf(cleaned.charAt(pos++));
        const e = CUSTOM_ALPHABET.indexOf(cleaned.charAt(pos++));
        result += String.fromCharCode(o << 2 | r >> 4);
        if (i !== 64) result += String.fromCharCode((15 & r) << 4 | i >> 2);
        if (e !== 64) result += String.fromCharCode((3 & i) << 6 | e);
    }
    return result;
}

function extractF1(src) {
    const match = src.match(/f1\s*=\s*\[([\s\S]*?)\],\s*s1\s*=/);
    if (!match) throw new Error('Could not locate f1 array');
    const result = [];
    const re = /"([^"]*?)"/g;
    let m;
    while ((m = re.exec(match[1])) !== null) result.push(m[1]);
    return result;
}

function extractS1(src) {
    const match = src.match(/s1\s*=\s*\[([\s\S]*?)\],\s*j\s*=\s*window/);
    if (!match) throw new Error('Could not locate s1 array');
    const W1=(n,t)=>5*(t&n)+5*(t&~n)-1*t-3*~(t&~t)+3*~(t|n)+3*~(t|~n);
    const T1=(n,t)=>-1*(n&t)-6*(n&~t)+2*n+5*~(n&t)-5*~(n|t)-5*~(n|~t);
    const M1=(n,t)=>2*(n&t)+2*(n&~t)-1*(n|t)-1*~(n&t)+1*~(n|t)+3*~(n|~t);
    const O1=(n,t)=>2*(n&~t)-1*(n^t)+1*~(n&~t)-1*~(n|t);
    const P1=(n,t)=>1*(n&t)-7*(n&~t)+4*~(n|t)+12*~(n|~t)-11*~n+7*~t;
    const d1=(n,t)=>2*(t&n)+2*(t&~n)-1*(t^n)-2*~(t|n)+2*~t;
    const B1=(n,t)=>2*(t&n)+1*(t^n)+7*~(t|n)+7*~(t|~n)-7*~t;
    return eval(`[${match[1]}]`);
}

// ═══════════════════════════════════════════════════════════════════════════════
// AST helpers
// ═══════════════════════════════════════════════════════════════════════════════

function evaluateArgument(node) {
    if (t.isNumericLiteral(node)) return node.value;
    if (t.isUnaryExpression(node) && node.operator === '-' && t.isNumericLiteral(node.argument))
        return -node.argument.value;
    return null;
}

function getStaticValue(node) {
    if (t.isNumericLiteral(node)) return node.value;
    if (t.isStringLiteral(node)) return node.value;
    if (t.isBooleanLiteral(node)) return node.value;
    if (t.isNullLiteral(node)) return null;
    if (t.isIdentifier(node) && node.name === 'undefined') return undefined;
    if (t.isUnaryExpression(node)) {
        const arg = getStaticValue(node.argument);
        if (arg === null && !t.isNullLiteral(node.argument)) return '__FAIL__';
        if (arg === '__FAIL__') return '__FAIL__';
        switch (node.operator) {
            case '-': return -arg;
            case '+': return +arg;
            case '~': return ~arg;
            case '!': return !arg;
            case 'void': return undefined;
            default: return '__FAIL__';
        }
    }
    if (t.isBinaryExpression(node)) return tryEvalBinary(node);
    return '__FAIL__';
}

function tryEvalBinary(node) {
    if (!t.isBinaryExpression(node)) return '__FAIL__';
    const l = getStaticValue(node.left);
    const r = getStaticValue(node.right);
    if (l === '__FAIL__' || r === '__FAIL__') return '__FAIL__';
    switch (node.operator) {
        case '+': return l + r;
        case '-': return l - r;
        case '*': return l * r;
        case '/': return r !== 0 ? l / r : '__FAIL__';
        case '%': return r !== 0 ? l % r : '__FAIL__';
        case '|': return l | r;
        case '&': return l & r;
        case '^': return l ^ r;
        case '<<': return l << r;
        case '>>': return l >> r;
        case '>>>': return l >>> r;
        case '==': return l == r;
        case '===': return l === r;
        case '!=': return l != r;
        case '!==': return l !== r;
        case '<': return l < r;
        case '>': return l > r;
        case '<=': return l <= r;
        case '>=': return l >= r;
        default: return '__FAIL__';
    }
}

function makeNode(val) {
    if (typeof val === 'string') return t.stringLiteral(val);
    if (typeof val === 'number') {
        if (val < 0) return t.unaryExpression('-', t.numericLiteral(-val));
        return t.numericLiteral(val);
    }
    if (typeof val === 'boolean') return t.booleanLiteral(val);
    if (val === null) return t.nullLiteral();
    if (val === undefined) return t.identifier('undefined');
    return null;
}

function isValidIdentifier(name) {
    if (typeof name !== 'string' || name.length === 0) return false;
    return /^[a-zA-Z_$][a-zA-Z0-9_$]*$/.test(name);
}

// ═══════════════════════════════════════════════════════════════════════════════
// MBA helpers
// ═══════════════════════════════════════════════════════════════════════════════

const mbaHelpers = {
    W1:(n,t)=>5*(t&n)+5*(t&~n)-1*t-3*~(t&~t)+3*~(t|n)+3*~(t|~n),
    T1:(n,t)=>-1*(n&t)-6*(n&~t)+2*n+5*~(n&t)-5*~(n|t)-5*~(n|~t),
    M1:(n,t)=>2*(n&t)+2*(n&~t)-1*(n|t)-1*~(n&t)+1*~(n|t)+3*~(n|~t),
    O1:(n,t)=>2*(n&~t)-1*(n^t)+1*~(n&~t)-1*~(n|t),
    P1:(n,t)=>1*(n&t)-7*(n&~t)+4*~(n|t)+12*~(n|~t)-11*~n+7*~t,
    d1:(n,t)=>2*(t&n)+2*(t&~n)-1*(t^n)-2*~(t|n)+2*~t,
    B1:(n,t)=>2*(t&n)+1*(t^n)+7*~(t|n)+7*~(t|~n)-7*~t,
    h1:(n,t)=>-2*(t&~n)+3*~(t&n)+1*~(t&~n)-4*~(t|n)-4*~(t|~n),
    v1:(n,t)=>8*(n&t)+6*(n&~t)-6*~(n&~n)+5*~(n|t)+7*~(n|~t)+1*~t,
    l1:(n,t)=>4*(t|n)-3*(~t&n)+6*~(t|n)-2*~(t^n)-~n-(t|~n)-~t+1,
    w1:(n,t)=>-1*(t&n)-1*(t&~n)+1*(t^n)+2*n-2*~(t|~n),
    X1:(n,t)=>-1*(t&n)-1*(t|~n)+2*t+1*~(t|n)-1*~(t|~n),
    m1:(n,t)=>-15*(t&n)-15*(t&~n)+5*t+11*~(t&~t)-11*~(t|n)-11*~(t|~n),
    p1:(n,t)=>-2*(n&t)-1*(n&~t)+2*n-1*~(n|t)-2*~(n|~t)+1*~n,
    g1:(n,t)=>-6*(n&t)-6*(n&~t)+1*t+7*~(n&~n)-7*~(n|t)-7*~(n|~t),
    S1:(n,t)=>13*(t&n)+8*(t&~n)-7*(t|n)-6*~(t^n)+6*~(t|n)+6*~(t|~n),
    R1:(n,t)=>-7*(t&n)-6*(t&~n)+7*(t|~n)-11*~(t|n)-5*~(t|~n)+4*~t,
    N1:(n,t)=>2*(n&t)+1*(n&~t)-1*~(n&~n)+2*~(n|t)+3*~(n|~t)-1*~n,
    k1:(n,t)=>-2*(n&t)+1*~(n&~n)+3*~(n&~t)-4*~(n|t)-3*~(n|~t),
    G1:(n,t)=>4*(n|t)-(n&~t)+4*~(n|t)-(n|~t)-~n-(~n|t)-~(n&t),
    Q1:(n,t)=>-3*(n&t)-8*(n&~t)+4*(n^t)+4*(n|~t)-4*~(n|t)-3*~(n|~t),
    A1:(n,t)=>-2*(n&t)-1*(n&~t)+2*(n^t)+2*t-5*~(n|~t),
    x1:(n,t)=>7*(t&n)-6*n+2*~(t|n)+9*~(t|~n)-2*~t,
    O:(n,t)=>1*(t&n)-2*(t&~n)-2*~(t|n)+1*~(t|~n)-1*~t+3*~n,
};

// ═══════════════════════════════════════════════════════════════════════════════
// Phase 0: Extract string tables
// ═══════════════════════════════════════════════════════════════════════════════

const f1 = extractF1(source);
const s1 = extractS1(source);
console.log(`[Phase 0] f1: ${f1.length}, s1: ${s1.length}`);

// ═══════════════════════════════════════════════════════════════════════════════
// Phase 1: Parse
// ═══════════════════════════════════════════════════════════════════════════════

console.log('[Phase 1] Parsing...');
const ast = parser.parse(source, {
    sourceType: 'script',
    allowReturnOutsideFunction: true,
    errorRecovery: true,
});

// ═══════════════════════════════════════════════════════════════════════════════
// Phase 2: Inline A(n) and J(n)
// ═══════════════════════════════════════════════════════════════════════════════

function inlineStringLookups() {
    let count = 0;
    traverse(ast, {
        CallExpression(path) {
            const { callee, arguments: args } = path.node;
            if (!t.isIdentifier(callee) || args.length !== 1) return;
            const name = callee.name;
            if (name !== 'A' && name !== 'J') return;
            const argVal = evaluateArgument(args[0]);
            if (argVal === null) return;
            const idx = argVal | 0;
            const decoded = name === 'J' ? decodeJ(idx, f1) : decodeA(idx, s1);
            if (decoded === undefined) return;
            const node = makeNode(decoded);
            if (node) { path.replaceWith(node); count++; }
        },
    });
    return count;
}

let total = 0, pass;
console.log('[Phase 2] Inlining A()/J()...');
pass = inlineStringLookups();
total += pass;
console.log(`  → Pass 1: ${pass}`);

// ═══════════════════════════════════════════════════════════════════════════════
// Phase 3: Resolve MBA helpers
// ═══════════════════════════════════════════════════════════════════════════════

function resolveMBA() {
    let count = 0;
    traverse(ast, {
        CallExpression(path) {
            const { callee, arguments: args } = path.node;
            if (!t.isIdentifier(callee) || !mbaHelpers[callee.name] || args.length !== 2) return;
            const a = getStaticValue(args[0]);
            const b = getStaticValue(args[1]);
            if (a === '__FAIL__' || b === '__FAIL__') return;
            const result = mbaHelpers[callee.name](a, b);
            if (!Number.isFinite(result)) return;
            path.replaceWith(t.numericLiteral(result));
            count++;
        },
    });
    return count;
}

console.log('[Phase 3] MBA helpers...');
pass = resolveMBA();
console.log(`  → ${pass}`);

// Second pass inlining after MBA resolution
pass = inlineStringLookups();
if (pass > 0) console.log(`  → Extra inlines: ${pass}`);
total += pass;

// ═══════════════════════════════════════════════════════════════════════════════
// Phase 3b: Replace j → window early (j = window alias used throughout)
// ═══════════════════════════════════════════════════════════════════════════════

let jCount = 0;
console.log('[Phase 3b] j → window...');
traverse(ast, {
    Identifier(path) {
        if (path.node.name !== 'j') return;
        if (path.parent && t.isMemberExpression(path.parent) && path.parent.object === path.node) {
            path.node.name = 'window';
            jCount++;
        }
    },
});
console.log(`  → ${jCount}`);

// ═══════════════════════════════════════════════════════════════════════════════
// Phase 4: Resolve j.Number(x), j.Math.floor/ceil(x), j.parseInt(x)
// ═══════════════════════════════════════════════════════════════════════════════

let mathCount = 0;
console.log('[Phase 4] Resolving j.Number/Math/parseInt...');

const mathResolvers = {
    'Number': (x) => Number(x),
    'parseInt': (x) => parseInt(x),
};
const mathMethodResolvers = {
    'floor': (x) => Math.floor(x),
    'ceil': (x) => Math.ceil(x),
    'round': (x) => Math.round(x),
    'abs': (x) => Math.abs(x),
    'atan2': null,
    'cos': null,
};

traverse(ast, {
    CallExpression(path) {
        const { callee, arguments: args } = path.node;
        if (args.length < 1) return;
        const argVal = getStaticValue(args[0]);
        if (argVal === '__FAIL__' || typeof argVal !== 'number') return;

        if (t.isMemberExpression(callee)) {
            const obj = callee.object;
            const prop = callee.property;

            const isJorWindow = (n) => t.isIdentifier(n, { name: 'j' }) || t.isIdentifier(n, { name: 'window' });
            const getPropName = (node, memberExpr) => {
                if (t.isIdentifier(node) && !memberExpr.computed) return node.name;
                if (t.isStringLiteral(node)) return node.value;
                return null;
            };

            const propName = getPropName(prop, callee);

            if (isJorWindow(obj) && propName) {
                const fn = mathResolvers[propName];
                if (fn) {
                    const result = fn(argVal);
                    if (Number.isFinite(result)) {
                        path.replaceWith(makeNode(result));
                        mathCount++;
                        return;
                    }
                }
            }

            if (t.isMemberExpression(obj)) {
                const grandObj = obj.object;
                const grandProp = obj.property;
                const grandPropName = getPropName(grandProp, obj);
                if (isJorWindow(grandObj) && grandPropName === 'Math' && propName) {
                    const fn = mathMethodResolvers[propName];
                    if (fn) {
                        const result = fn(argVal);
                        if (Number.isFinite(result)) {
                            path.replaceWith(makeNode(result));
                            mathCount++;
                            return;
                        }
                    }
                }
            }
        }
    },
});
console.log(`  → ${mathCount}`);

// ═══════════════════════════════════════════════════════════════════════════════
// Phase 5: Constant folding (multi-pass)
// ═══════════════════════════════════════════════════════════════════════════════

let foldedTotal = 0;
console.log('[Phase 5] Constant folding...');

for (let iter = 0; iter < 3; iter++) {
    let foldedCount = 0;
    traverse(ast, {
        BinaryExpression(path) {
            const result = getStaticValue(path.node);
            if (result === '__FAIL__') return;
            const node = makeNode(result);
            if (node) { path.replaceWith(node); foldedCount++; }
        },
        UnaryExpression(path) {
            const { operator, argument } = path.node;
            if (operator === '!' && t.isNumericLiteral(argument)) {
                path.replaceWith(t.booleanLiteral(!argument.value)); foldedCount++;
            } else if (operator === '!' && t.isBooleanLiteral(argument)) {
                path.replaceWith(t.booleanLiteral(!argument.value)); foldedCount++;
            } else if (operator === 'void' && t.isNumericLiteral(argument) && argument.value === 0) {
                path.replaceWith(t.identifier('undefined')); foldedCount++;
            } else if (operator === '~' && t.isNumericLiteral(argument)) {
                path.replaceWith(makeNode(~argument.value)); foldedCount++;
            } else if (operator === '-' && t.isUnaryExpression(argument) && argument.operator === '-') {
                path.replaceWith(argument.argument); foldedCount++;
            }
        },
    });
    foldedTotal += foldedCount;
    if (foldedCount === 0) break;
}
console.log(`  → ${foldedTotal}`);

// ═══════════════════════════════════════════════════════════════════════════════
// Phase 6: Bracket → dot notation
// ═══════════════════════════════════════════════════════════════════════════════

let dotCount = 0;
console.log('[Phase 6] Bracket → dot...');
traverse(ast, {
    MemberExpression(path) {
        if (!path.node.computed || !t.isStringLiteral(path.node.property)) return;
        const prop = path.node.property.value;
        if (isValidIdentifier(prop)) {
            path.node.computed = false;
            path.node.property = t.identifier(prop);
            dotCount++;
        }
    },
});
console.log(`  → ${dotCount}`);

// ═══════════════════════════════════════════════════════════════════════════════
// Phase 7: Normalize typeof
// ═══════════════════════════════════════════════════════════════════════════════

let typeofCount = 0;
console.log('[Phase 7] Normalize typeof...');
traverse(ast, {
    BinaryExpression(path) {
        const { operator, left, right } = path.node;
        if (!['==', '===', '!=', '!=='].includes(operator)) return;
        if (t.isStringLiteral(left) && t.isUnaryExpression(right) && right.operator === 'typeof') {
            path.node.left = right;
            path.node.right = left;
            typeofCount++;
        }
    },
});
console.log(`  → ${typeofCount}`);

// ═══════════════════════════════════════════════════════════════════════════════
// Phase 8: Remove dead code — expression statements that are just
// j.Number(x), j.Math.ceil(x), etc. with no side effects
// ═══════════════════════════════════════════════════════════════════════════════

let deadCount = 0;
console.log('[Phase 8] Removing dead code...');
traverse(ast, {
    ExpressionStatement(path) {
        const expr = path.node.expression;
        if (t.isNumericLiteral(expr) || t.isStringLiteral(expr) || t.isBooleanLiteral(expr)) {
            path.remove();
            deadCount++;
            return;
        }
        // j.Number(x), j.Math.floor(x) as standalone statements
        if (t.isCallExpression(expr) && t.isMemberExpression(expr.callee)) {
            const callee = expr.callee;
            // j.Number(x) or j.parseInt(x)
            if ((t.isIdentifier(callee.object, { name: 'j' }) || t.isIdentifier(callee.object, { name: 'window' })) && t.isIdentifier(callee.property) &&
                ['Number', 'parseInt'].includes(callee.property.name) && expr.arguments.length === 1) {
                const argVal = getStaticValue(expr.arguments[0]);
                if (argVal !== '__FAIL__') {
                    path.remove();
                    deadCount++;
                    return;
                }
            }
            // j.Math.xxx(x)
            if (t.isMemberExpression(callee.object) &&
                t.isIdentifier(callee.object.object, { name: 'j' }) &&
                t.isIdentifier(callee.object.property, { name: 'Math' }) &&
                t.isIdentifier(callee.property) && expr.arguments.length === 1) {
                const argVal = getStaticValue(expr.arguments[0]);
                if (argVal !== '__FAIL__') {
                    path.remove();
                    deadCount++;
                    return;
                }
            }
        }
        // Comma expressions where both sides are dead
        if (t.isSequenceExpression(expr)) {
            const exprs = expr.expressions;
            const allDead = exprs.every(e => {
                if (t.isNumericLiteral(e) || t.isStringLiteral(e) || t.isBooleanLiteral(e)) return true;
                if (t.isCallExpression(e) && t.isMemberExpression(e.callee)) {
                    const c = e.callee;
                    if ((t.isIdentifier(c.object, { name: 'j' }) || t.isIdentifier(c.object, { name: 'window' })) && t.isIdentifier(c.property) &&
                        ['Number', 'parseInt'].includes(c.property.name)) return true;
                    if (t.isMemberExpression(c.object) &&
                        (t.isIdentifier(c.object.object, { name: 'j' }) || t.isIdentifier(c.object.object, { name: 'window' })) &&
                        t.isIdentifier(c.object.property, { name: 'Math' })) return true;
                }
                return false;
            });
            if (allDead) {
                path.remove();
                deadCount++;
            }
        }
    },
});
console.log(`  → ${deadCount}`);

// ═══════════════════════════════════════════════════════════════════════════════
// Phase 9: Generate
// ═══════════════════════════════════════════════════════════════════════════════

console.log('[Phase 10] Generating...');
const { code: outputCode } = generate(ast, {
    comments: true,
    compact: false,
    concise: false,
});

fs.writeFileSync(outputFile, outputCode, 'utf-8');
console.log(`\nDone: ${outputFile}`);
console.log(`  ${source.length} → ${outputCode.length} chars`);

const tableFile = outputFile.replace('.js', '_strings.json');
const tables = { f1_decoded: {}, s1_decoded: {} };
for (let i = 0; i < f1.length; i++) tables.f1_decoded[i] = decodeJ(i, f1);
for (let i = 0; i < s1.length; i++) {
    const raw = s1[i];
    const decoded = decodeA(i, s1);
    tables.s1_decoded[i] = typeof raw === 'string' ? { encoded: raw, decoded } : { value: raw };
}
fs.writeFileSync(tableFile, JSON.stringify(tables, null, 2), 'utf-8');
console.log(`  Strings: ${tableFile}`);

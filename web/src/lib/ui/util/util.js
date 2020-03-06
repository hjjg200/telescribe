
export function colorify(str) {
    let hash = 0;
    for(let i = 0; i < str.length; i++) {
       hash = str.charCodeAt(i) + ((hash << 5) - hash);
    }
    let hue = (hash & 0x000FFF) % 120;
    hue = hue + (hue > 55 ? 290 : 200);
    hue = hue % 360;
    let sat = (hash & 0xFFF000 >> 12) % 25;
    sat = sat + (sat > 5 ? 70 : 25);
    return `hsl(${hue}, ${sat}%, 50%)`;
}
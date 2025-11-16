pub fn remove_overstrike(input: &str) -> String {
    let runes: Vec<char> = input.chars().collect();
    let mut output = String::with_capacity(runes.len());
    let mut i = 0;
    while i < runes.len() {
        if i + 2 < runes.len() && runes[i + 1] == '\u{0008}' {
            output.push(runes[i + 2]);
            i += 3;
        } else {
            output.push(runes[i]);
            i += 1;
        }
    }
    output
}

const go = new Go();
let emu;
let chip8;

const chip8Controls = [
    'KeyX',
    'Digit1',
    'Digit2',
    'Digit3',
    'KeyQ',
    'KeyW',
    'KeyE',
    'KeyA',
    'KeyS',
    'KeyD',
    'KeyZ',
    'KeyC',
    'Digit4',
    'KeyR',
    'KeyF',
    'KeyV',
];

let keys = 0;

window.addEventListener("keyup", (e) => {
    if (chip8Controls.includes(e.code)) {
        key = chip8Controls.indexOf(e.code);
        keys &= ~(1 << key);
    }
});

window.addEventListener("keydown", (e) => {
    if (chip8Controls.includes(e.code)) {
        key = chip8Controls.indexOf(e.code);
        keys |= 1 << key;
    }
});

function render() {
    ptr = chip8.GetFrame();
    const image = new Uint8ClampedArray(
        chip8.memory.buffer,
        ptr,
        32 * 64 * 4
    );

    const canvas = document.getElementById("chip8");
    const context = canvas.getContext("2d");
    const pixelData = new ImageData(image, 64, 32);
    context.putImageData(pixelData, 0, 0);
}

function RAFCallback() {
    chip8.SetKeys(keys);
    chip8.Step();
    chip8.Step();
    chip8.Step();
    chip8.Step();
    chip8.Step();
    chip8.Step();
    chip8.Step();
    chip8.Step();
    render();
}
let id;
WebAssembly.instantiateStreaming(fetch("chip8.wasm"), go.importObject).then(
    (result) => {
        go.run(result.instance);
        chip8 = result.instance.exports;

        let romName = "space_invaders.ch8"
        fetch(romName)
            .then((response) => response.arrayBuffer())
            .then((rom) => {
                let ptr = chip8.malloc(rom.byteLength);

                (new Uint8Array(chip8.memory.buffer, ptr)).set(new Uint8Array(rom));

                chip8.InitChip8(ptr, rom.byteLength, rom.byteLength);

                console.log(chip8.memory.buffer)

                id = setInterval(RAFCallback, 16)
            });
    }
);

newRom = document.getElementById("newRom");
newRom.onclick = () => {
    fetch("mySnake.ch8")
        .then((response) => response.arrayBuffer())
        .then((rom) => {
            let ptr = chip8.malloc(rom.byteLength);

            (new Uint8Array(chip8.memory.buffer, ptr)).set(new Uint8Array(rom));

            chip8.InitChip8(ptr, rom.byteLength, rom.byteLength);
        });
}
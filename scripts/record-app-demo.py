"""Record actual Android UI interactions and synchronize local Russian narration."""
import json
import math
import re
import subprocess
import time
import threading
import wave
import xml.etree.ElementTree as ET
from pathlib import Path

OUT = Path('artifacts/app-demo').resolve()
ADB = ['adb', '-s', 'emulator-5554']


def adb(*args):
    return subprocess.check_output(ADB + list(args), stderr=subprocess.STDOUT)


def tree():
    adb('shell', 'uiautomator', 'dump', '/sdcard/appvideo-ui.xml')
    data = adb('exec-out', 'cat', '/sdcard/appvideo-ui.xml')
    return ET.fromstring(data)


def tap(text):
    root = tree()
    nodes = [n for n in root.iter('node') if n.get('text') == text]
    if len(nodes) != 1:
        raise RuntimeError(f'Expected one control: {text}; found {len(nodes)}')
    x1, y1, x2, y2 = map(int, re.findall(r'\d+', nodes[0].get('bounds')))
    adb('shell', 'input', 'tap', str((x1+x2)//2), str((y1+y2)//2))
    time.sleep(.8)


def require(text):
    root = tree()
    content = '\n'.join(n.get('text', '') for n in root.iter('node'))
    if text not in content:
        raise RuntimeError(f'Missing expected result {text}: {content}')
    return ET.tostring(root, encoding='utf-8')


chapters = json.loads((OUT/'chapters.json').read_text(encoding='utf-8-sig'))
for i, chapter in enumerate(chapters):
    with wave.open(str(OUT/f'voice-{i}.wav')) as audio:
        duration = audio.getnframes()/audio.getframerate()
    # UI dumps take time; leave a brief lead-in before narration begins.
    limit = math.ceil(duration) + 9
    recording = subprocess.Popen(['ffmpeg', '-y', '-v', 'error', '-f', 'image2pipe',
        '-framerate', '3', '-i', '-', '-vf', 'scale=720:1600,setsar=1',
        '-c:v', 'libx264', '-preset', 'ultrafast', '-pix_fmt', 'yuv420p',
        str(OUT/f'raw-{i}.mp4')], stdin=subprocess.PIPE)
    started = time.monotonic()
    def capture():
        frame = adb('exec-out', 'screencap', '-p')
        written = 0
        while written < limit * 3:
            frame = adb('exec-out', 'screencap', '-p')
            target = min(limit * 3, max(written + 1, int((time.monotonic()-started)*3)))
            while written < target:
                recording.stdin.write(frame)
                written += 1
            time.sleep(max(0, started + written/3 - time.monotonic()))
        recording.stdin.close()
    capture_thread = threading.Thread(target=capture)
    capture_thread.start()
    time.sleep(1)
    action = chapter['action']
    expected = None
    if action in ('fraud', 'legal'):
        tap('demo-' + action)
        tap('Проверить защищённое действие')
        expected = 'hold' if action == 'fraud' else 'step_up'
    elif action.startswith('state-'):
        tap('Проверить состояние услуги')
        expected = 'Изменений: 0' if action == 'state-fraud' else 'Изменений: 1'
    elif action == 'confirm':
        tap('Тест: подтверждаю sink-a')
        expected = 'allow'
    if expected:
        xml = require(expected)
        (OUT/f'ui-{i}.xml').write_bytes(xml)
    (OUT/f'screen-{i}.png').write_bytes(adb('exec-out', 'screencap', '-p'))
    remaining = limit + 2 - (time.monotonic() - started)
    if remaining > 0:
        time.sleep(remaining)
    capture_thread.join()
    assert recording.wait(timeout=15) == 0
    subprocess.run(['ffmpeg', '-y', '-v', 'error', '-i', str(OUT/f'raw-{i}.mp4'),
        '-i', str(OUT/f'voice-{i}.wav'), '-filter_complex',
        '[0:v]scale=720:1600,fps=30,setsar=1[v];[1:a]adelay=6500:all=1,apad[a]',
        '-map', '[v]', '-map', '[a]', '-t', str(limit), '-c:v', 'libx264',
        '-preset', 'fast', '-crf', '21', '-pix_fmt', 'yuv420p', '-c:a', 'aac',
        '-ar', '48000', '-ac', '2', str(OUT/f'chapter-{i}.mp4')], check=True)
    print(f'Chapter {i+1}/{len(chapters)}: {chapter["title"]}', flush=True)
(OUT/'concat.txt').write_text(''.join(f"file 'chapter-{i}.mp4'\n" for i in range(len(chapters))), encoding='utf-8')
subprocess.run(['ffmpeg', '-y', '-v', 'error', '-f', 'concat', '-safe', '0', '-i',
    str(OUT/'concat.txt'), '-c', 'copy', '-movflags', '+faststart', str(OUT/'antifraud-app-demo.mp4')], check=True)
print(OUT/'antifraud-app-demo.mp4')

from renderer.utils import render_template_from_string

def render(context, params):
    rat = 'vertical-rat.gif' if params.get('direction') == 'vertical' else 'horizontal-rat.gif'
    return render_template_from_string(
        """
        <img src="/-/static/rat/{{ rat }}" alt="这只慢吞吞的耗子在神秘地移动"  width="250" />
        <audio autoplay loop>
            <source src="/-/static/rat/rat.mp3" type="audio/mpeg">
            您的浏览器不支持音频元素。
        </audio>
        """,
        rat=rat
    )
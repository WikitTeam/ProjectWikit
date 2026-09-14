
    <script>
    (function(){
        let lastHeight = -1;
        let settling = 0;
        function measure() {
            const body = document.body;
            const html = document.documentElement;
            return Math.max(body && body.scrollHeight, body && body.offsetHeight, html.offsetHeight, body && body.getBoundingClientRect().height);
        }
        function report(force) {
            const height = measure();
            if (force || lastHeight !== height) {
                parent.postMessage({type: 'iframe-change-height', payload: { height, id: %s } }, '*');
                lastHeight = height;
            }
        }
        function doFrame() {
            window.requestAnimationFrame(doFrame);
            report(false);
        }
        doFrame();
        const settle = setInterval(function(){
            report(true);
            if (++settling > 20) clearInterval(settle);
        }, 100);
    })();
    const apiHandler = {
        get(target, name) {
            return (async (...args) => {
                const data = {
                    type: "ApiCall",
                    target: name,
                    callId: Math.random(),
                    args
                }
                window.parent.postMessage(data, "*");
                let result;
                const responsePromise = new Promise((resolve) => {
                    const listener = (e) => {
                        if (!e.data.hasOwnProperty("type") || 
                            !e.data.hasOwnProperty("target") || 
                            !e.data.hasOwnProperty("callId") || 
                            !e.data.hasOwnProperty("response") || 
                            e.data.type !== "ApiResponse" ||
                            e.data.callId !== data.callId)
                            return;
                        window.removeEventListener("message", listener);
                        result = e.data.response;
                        resolve();
                    }
                    window.addEventListener("message", listener);
                });
                await responsePromise;
                return result;
            });
        }
    }
    const api = new Proxy({}, apiHandler);
    </script>
    <style>
      html, body {
        margin: 0;
        padding: 0;
        overflow-y: hidden;
      }
    </style> 
    
/**
 * Copyright (c) 2020-2026, The MathWorks, Inc. All rights reserved.
 */

window.onload = function() {
    window.location.href = document.getElementById("redirect-url").href;
    setTimeout(function() { window.close(); }, 1000);
};

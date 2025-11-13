(function() {
  console.log('Mermaid-zoom plugin loaded');

  // Create an SVG that contains the entire mermaid container for zoom
  const createContainerSvg = (container) => {
    // This is a fallback for closed shadow DOM - create an SVG wrapper
    const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
    const foreignObject = document.createElementNS("http://www.w3.org/2000/svg", "foreignObject");
    
    // Get container dimensions
    const rect = container.getBoundingClientRect();
    svg.setAttribute("width", rect.width);
    svg.setAttribute("height", rect.height);
    svg.setAttribute("viewBox", `0 0 ${rect.width} ${rect.height}`);
    
    foreignObject.setAttribute("x", "0");
    foreignObject.setAttribute("y", "0");
    foreignObject.setAttribute("width", rect.width);
    foreignObject.setAttribute("height", rect.height);
    
    // Clone the container
    const clonedContainer = container.cloneNode(true);
    foreignObject.appendChild(clonedContainer);
    svg.appendChild(foreignObject);
    
    return svg;
  };

  const enhance = (diagramDiv) => {
    if (diagramDiv.dataset.enhanced) return;
    diagramDiv.dataset.enhanced = 'true';
    diagramDiv.style.cursor = 'zoom-in';
    
    diagramDiv.addEventListener("click", (e) => {
      e.preventDefault();
      e.stopImmediatePropagation();
      
      const svg = diagramDiv.querySelector("svg");
      if (svg) {
        createLightbox(svg);
      } else {
        console.warn('No SVG found for zoom interaction');
      }
    }, true);
  };

  const createLightbox = (svg) => {
    // Prevent multiple lightboxes
    if (document.querySelector('.mermaid-lightbox-overlay')) {
      return;
    }

    const overlay = document.createElement('div');
    overlay.className = 'mermaid-lightbox-overlay';
    overlay.setAttribute('data-mermaid-lightbox', 'true');

    const clonedSvg = svg.cloneNode(true);
    clonedSvg.className = 'mermaid-lightbox-svg';
    clonedSvg.style.maxWidth = '90%';
    clonedSvg.style.maxHeight = '90%';
    clonedSvg.style.cursor = 'grab';

    const closeBtn = document.createElement('button');
    closeBtn.className = 'mermaid-lightbox-close';
    closeBtn.innerHTML = '&times;';

    overlay.appendChild(clonedSvg);
    overlay.appendChild(closeBtn);
    document.body.appendChild(overlay);

    // Add interaction
    addInteraction(overlay, clonedSvg);

    const closeLightbox = () => {
      if (document.body.contains(overlay)) {
        document.body.removeChild(overlay);
      }
      document.removeEventListener('keydown', onKeydown);
    };

    const onKeydown = (e) => {
      if (e.key === 'Escape') closeLightbox();
    };

    overlay.addEventListener('click', closeLightbox);
    closeBtn.addEventListener('click', closeLightbox);
    clonedSvg.addEventListener('click', e => e.stopPropagation());
    document.addEventListener('keydown', onKeydown);
  };

  const addInteraction = (container, svg) => {
    let scale = 1, pointX = 0, pointY = 0;
    let isDragging = false, startPos = { x: 0, y: 0 };

    svg.style.transformOrigin = 'center';
    svg.style.transition = 'transform 0.1s ease-out';

    const setTransform = () => {
      svg.style.transform = `translate(${pointX}px, ${pointY}px) scale(${scale})`;
    };

    svg.addEventListener('mousedown', e => {
      e.stopPropagation();
      isDragging = true;
      startPos = { x: e.clientX - pointX, y: e.clientY - pointY };
      svg.style.cursor = 'grabbing';
    });

    window.addEventListener('mousemove', e => {
      if (!isDragging) return;
      pointX = e.clientX - startPos.x;
      pointY = e.clientY - startPos.y;
      setTransform();
    });

    window.addEventListener('mouseup', () => {
      isDragging = false;
      svg.style.cursor = 'grab';
    });

    container.addEventListener('wheel', e => {
      e.preventDefault();
      const delta = e.deltaY < 0 ? 0.1 : -0.1;
      scale = Math.max(0.2, Math.min(10, scale + delta));
      setTransform();
    }, { passive: false });
  };

  const scanAndEnhance = () => {
    const containers = document.querySelectorAll('.mermaid:not([data-mermaid-lightbox])');
    
    containers.forEach(container => {
      const svg = container.querySelector('svg');
      if (svg) {
        enhance(container);
      }
    });
  };

  const waitForMermaid = () => {
    if (typeof mermaid === 'undefined') {
      setTimeout(waitForMermaid, 100);
      return;
    }
    
    // Initial scan
    scanAndEnhance();
    
    // Monitor for new content
    let attempts = 0;
    const maxAttempts = 20;
    
    const waitForRendering = () => {
      attempts++;
      scanAndEnhance();
      
      if (attempts < maxAttempts) {
        setTimeout(waitForRendering, 200);
      }
    };
    
    waitForRendering();
  };

  // Initialize when page loads
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', waitForMermaid);
  } else {
    waitForMermaid();
  }

  // Also handle live reload
  document.addEventListener('DOMContentLoaded', () => {
    setTimeout(waitForMermaid, 100);
  });
})(); 
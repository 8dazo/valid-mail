import { useId } from "react";
import { motion, useReducedMotion } from "motion/react";
import type { ReactNode } from "react";
import {
  ArrowRight,
  ArrowUpRight,
  Braces,
  Check,
  ChevronDown,
  Copy,
  Download,
  FileText,
  GitFork,
  LockKeyhole,
  MailCheck,
  Sparkles,
} from "lucide-react";

type OpenTool = (id: string) => void;

function ToolkitArt({ kind }: { kind: "check" | "tidy" | "make" }) {
  const id = useId().replaceAll(":", "");
  return (
    <svg className="toolkit-art" viewBox="0 0 320 210" aria-hidden="true">
      <defs>
        <linearGradient id={`${id}-paper`} x2="0.8" y2="1">
          <stop stopColor="#fff" />
          <stop offset="1" stopColor="#e5dcfb" />
        </linearGradient>
        <linearGradient id={`${id}-purple`} x2="0.8" y2="1">
          <stop stopColor="#ccb7ed" />
          <stop offset="1" stopColor="#8d70bd" />
        </linearGradient>
        <filter
          id={`${id}-shadow`}
          x="-30%"
          y="-30%"
          width="160%"
          height="180%"
        >
          <feDropShadow
            dx="0"
            dy="8"
            stdDeviation="4"
            floodColor="#725197"
            floodOpacity=".14"
          />
        </filter>
      </defs>
      <ellipse cx="162" cy="187" rx="100" ry="10" fill="#715492" opacity=".1" />
      {kind === "check" ? (
        <g className="toolkit-art-main" filter={`url(#${id}-shadow)`}>
          <rect
            x="88"
            y="35"
            width="145"
            height="112"
            rx="15"
            fill={`url(#${id}-paper)`}
            stroke="#b29bcf"
            strokeWidth="2"
            transform="rotate(-9 160 95)"
          />
          <path
            d="M66 99h184v70a14 14 0 0 1-14 14H80a14 14 0 0 1-14-14z"
            fill="#c3abeb"
            stroke="#9b7ebd"
            strokeWidth="2"
          />
          <path
            d="m67 102 91 58 91-58M67 176l60-49m122 49-60-49"
            fill="none"
            stroke="#9b7ebd"
            strokeWidth="2"
          />
          <circle
            cx="231"
            cy="151"
            r="31"
            fill="#d5edcb"
            stroke="#83ad7b"
            strokeWidth="2"
          />
          <path
            d="m216 151 10 10 19-22"
            fill="none"
            stroke="#598150"
            strokeWidth="5"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
          <path
            d="M127 60h57m-57 13h39"
            stroke="#baaad1"
            strokeWidth="4"
            strokeLinecap="round"
          />
        </g>
      ) : kind === "tidy" ? (
        <g className="toolkit-art-main" filter={`url(#${id}-shadow)`}>
          <rect
            x="117"
            y="45"
            width="125"
            height="145"
            rx="13"
            fill="#b9d5ee"
            stroke="#88accb"
            strokeWidth="2"
            transform="rotate(12 180 110)"
          />
          <rect
            x="76"
            y="28"
            width="125"
            height="145"
            rx="13"
            fill={`url(#${id}-paper)`}
            stroke="#b29bcf"
            strokeWidth="2"
            transform="rotate(-8 140 100)"
          />
          <path
            d="M111 62h49m-49 14h67m-67 14h38"
            stroke="#b7a4d1"
            strokeWidth="4"
            strokeLinecap="round"
          />
          <rect
            x="113"
            y="108"
            width="87"
            height="57"
            rx="12"
            fill={`url(#${id}-purple)`}
            stroke="#9474b9"
            strokeWidth="2"
          />
          <text
            x="156"
            y="147"
            textAnchor="middle"
            fill="#fff"
            fontSize="31"
            fontFamily="monospace"
          >
            {"{ }"}
          </text>
        </g>
      ) : (
        <g className="toolkit-art-main" filter={`url(#${id}-shadow)`}>
          <rect
            x="83"
            y="43"
            width="130"
            height="143"
            rx="18"
            fill="#ebcbe4"
            stroke="#bd95b8"
            strokeWidth="2"
            transform="rotate(-10 145 110)"
          />
          <rect
            x="104"
            y="28"
            width="130"
            height="143"
            rx="18"
            fill={`url(#${id}-paper)`}
            stroke="#b29bcf"
            strokeWidth="2"
          />
          <g fill="#967ab8">
            <path d="M122 46h32v32h-32zm47 0h32v32h-32zm-47 47h32v32h-32z" />
            <path d="M169 94h11v11h-11zm21 0h11v21h-11zm-21 22h22v10h-22zm21 10h12v12h-12z" />
          </g>
          <g fill="#f7f2ff">
            <path d="M130 54h16v16h-16zm47 0h16v16h-16zm-47 47h16v16h-16z" />
          </g>
          <circle
            cx="226"
            cy="153"
            r="29"
            fill="#f8ddb8"
            stroke="#cbaa82"
            strokeWidth="2"
          />
          <path
            d="m226 134 5 13 13 5-13 5-5 13-5-13-13-5 13-5z"
            fill="#b893cf"
          />
        </g>
      )}
      <path d="m47 49 4 8 8 4-8 4-4 8-4-8-8-4 8-4z" fill="#b899d7" />
      <circle cx="272" cy="79" r="6" fill="#e9b5ce" />
      <circle cx="54" cy="144" r="4" fill="#91bcb0" />
      <path
        d="m260 31 8 6m-7 2 6-9"
        stroke="#a2c6b3"
        strokeWidth="3"
        strokeLinecap="round"
      />
    </svg>
  );
}

/**
 * A tiny illustrated network of useful tools. The shapes are intentionally SVG,
 * so they stay crisp on retina screens without loading a second hero image.
 */
export function HeroSignalArt() {
  return (
    <svg
      className="hero-signal-art"
      viewBox="0 0 420 390"
      fill="none"
      aria-hidden="true"
    >
      <defs>
        <linearGradient id="heroSignalPaper" x1="0" y1="0" x2="1" y2="1">
          <stop stopColor="#ffffff" />
          <stop offset="1" stopColor="#faf3ff" />
        </linearGradient>
        <linearGradient id="heroSignalDisc" x1="0" y1="0" x2="1" y2="1">
          <stop stopColor="#c8b1f6" />
          <stop offset="1" stopColor="#8a72c3" />
        </linearGradient>
        <filter
          id="heroSignalShadow"
          x="-50%"
          y="-50%"
          width="200%"
          height="220%"
        >
          <feDropShadow
            dx="0"
            dy="15"
            stdDeviation="16"
            floodColor="#30234d"
            floodOpacity=".19"
          />
        </filter>
      </defs>
      <circle
        cx="210"
        cy="192"
        r="151"
        stroke="#fff"
        strokeOpacity=".45"
        strokeDasharray="3 9"
      />
      <circle cx="210" cy="192" r="115" stroke="#fff" strokeOpacity=".52" />
      <circle
        cx="210"
        cy="192"
        r="78"
        fill="#ffffff"
        fillOpacity=".12"
        stroke="#ffffff"
        strokeOpacity=".5"
      />
      <path
        className="hero-signal-route"
        d="M105 131C151 68 279 65 324 139S321 286 238 307 82 254 105 131Z"
        stroke="#fff"
        strokeOpacity=".7"
        strokeWidth="2"
        strokeDasharray="7 12"
      />
      <g className="hero-signal-center" filter="url(#heroSignalShadow)">
        <rect
          x="157"
          y="139"
          width="106"
          height="106"
          rx="33"
          fill="url(#heroSignalPaper)"
        />
        <rect x="164" y="146" width="92" height="92" rx="27" stroke="#e7dafa" />
        <path
          d="M181 189h55m-55 15h36"
          stroke="#c5b1e6"
          strokeWidth="7"
          strokeLinecap="round"
        />
        <circle cx="225" cy="174" r="19" fill="#e0f3e6" />
        <path
          d="m215 174 7 7 13-15"
          stroke="#4f916d"
          strokeWidth="4.5"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </g>
      <g
        className="hero-signal-card hero-signal-card-one"
        filter="url(#heroSignalShadow)"
      >
        <rect
          x="20"
          y="92"
          width="120"
          height="77"
          rx="18"
          fill="url(#heroSignalPaper)"
        />
        <rect x="35" y="110" width="28" height="28" rx="9" fill="#e1d5f9" />
        <path
          d="m41 119 8 6 8-6m-16 0v10h16v-10"
          stroke="#8568bb"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
        <path
          d="M73 117h50M73 128h32"
          stroke="#c7b9de"
          strokeWidth="5"
          strokeLinecap="round"
        />
        <rect x="35" y="146" width="88" height="7" rx="3.5" fill="#e9e1f4" />
      </g>
      <g
        className="hero-signal-card hero-signal-card-two"
        filter="url(#heroSignalShadow)"
      >
        <rect
          x="292"
          y="130"
          width="104"
          height="85"
          rx="18"
          fill="url(#heroSignalPaper)"
        />
        <rect x="307" y="145" width="74" height="55" rx="10" fill="#e9f4e8" />
        <path
          d="m322 171 12 10 20-23"
          stroke="#75a97d"
          strokeWidth="6"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
        <circle cx="363" cy="165" r="4" fill="#b3cdb4" />
      </g>
      <g
        className="hero-signal-card hero-signal-card-three"
        filter="url(#heroSignalShadow)"
      >
        <rect
          x="91"
          y="270"
          width="139"
          height="71"
          rx="18"
          fill="url(#heroSignalPaper)"
        />
        <rect x="105" y="283" width="39" height="42" rx="10" fill="#f4e1d9" />
        <path
          d="m127 291-8 13 8 13m-2-13h12"
          stroke="#b78875"
          strokeWidth="3"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
        <path
          d="M153 292h61m-61 12h46m-46 12h55"
          stroke="#d7c8e9"
          strokeWidth="5"
          strokeLinecap="round"
        />
      </g>
      <g className="hero-signal-spark">
        <path d="m334 57 7 16 16 7-16 7-7 16-7-16-16-7 16-7z" fill="#f9e2a9" />
        <circle cx="65" cy="225" r="7" fill="#d2edce" />
        <circle cx="282" cy="310" r="5" fill="#ebcde1" />
        <path d="m52 51 4 7 8 4-8 4-4 8-4-8-7-4 7-4z" fill="#ffebd2" />
      </g>
    </svg>
  );
}

export function LandingIntro({
  count,
  localCount,
  onOpen,
}: {
  count: number;
  localCount: number;
  onOpen: OpenTool;
}) {
  const reduced = useReducedMotion();
  return (
    <div className="studio-landing">
      <section
        className="studio-stats container"
        aria-label="The toolkit at a glance"
      >
        {[
          [`${count}`, "Everyday tools", "One place for the little things."],
          [
            `${localCount}`,
            "Work locally",
            "Your inputs stay in your browser.",
          ],
          ["0", "Accounts needed", "Skip the setup. Start creating."],
          ["100%", "Free to use", "A good shortcut for everyone."],
        ].map(([value, title, text]) => (
          <motion.div
            key={title}
            initial={reduced ? false : { opacity: 0, y: 18 }}
            whileInView={reduced ? undefined : { opacity: 1, y: 0 }}
            viewport={{ once: true, amount: 0.3 }}
            transition={{
              duration: 0.5,
              delay:
                [
                  "Everyday tools",
                  "Work locally",
                  "Accounts needed",
                  "Free to use",
                ].indexOf(title) * 0.06,
            }}
          >
            <strong>{value}</strong>
            <h3>{title}</h3>
            <p>{text}</p>
          </motion.div>
        ))}
      </section>
      <section
        className="studio-features container"
        id="how-it-works"
        aria-labelledby="features-title"
      >
        <div className="studio-section-heading">
          <div>
            <span className="studio-kicker">
              THOUGHTFUL TOOLS, BEAUTIFULLY SIMPLE
            </span>
            <h2 id="features-title">
              Everything you need
              <br />
              to keep things moving.
            </h2>
          </div>
          <p>
            From a messy bit of data to a finished little idea.
            <br />
            Pick a tool, bring your input, make it yours.
          </p>
        </div>
        <div className="studio-feature-grid">
          <motion.article
            className="studio-feature peach"
            initial={reduced ? false : { opacity: 0, y: 26 }}
            whileInView={reduced ? undefined : { opacity: 1, y: 0 }}
            viewport={{ once: true, amount: 0.15 }}
            transition={{ duration: 0.55, ease: "easeOut" }}
          >
            <div className="feature-mini editorial-feature-art">
              <img src="/images/editorial/feature-check.webp" alt="" loading="lazy" decoding="async" />
              <span className="editorial-feature-tag">
                <MailCheck size={14} /> Check an address
              </span>
            </div>
            <span className="feature-step">01 / CHECK</span>
            <h3>A little more certainty.</h3>
            <p>
              Check email addresses, inspect links and understand what’s behind
              your data.
            </p>
            <button onClick={() => onOpen("email")}>
              Check an email <ArrowUpRight size={16} />
            </button>
          </motion.article>
          <motion.article
            className="studio-feature sage"
            initial={reduced ? false : { opacity: 0, y: 26 }}
            whileInView={reduced ? undefined : { opacity: 1, y: 0 }}
            viewport={{ once: true, amount: 0.15 }}
            transition={{ duration: 0.55, ease: "easeOut" }}
          >
            <div className="feature-mini editorial-feature-art">
              <img src="/images/editorial/feature-tidy.webp" alt="" loading="lazy" decoding="async" />
              <span className="editorial-feature-tag">
                <Braces size={14} /> Tidy your data
              </span>
            </div>
            <span className="feature-step">02 / TIDY</span>
            <h3>Give the mess some shape.</h3>
            <p>
              Format, convert and preview. Turn the things you’re working on
              into something useful.
            </p>
            <button onClick={() => onOpen("json")}>
              Tidy some JSON <ArrowUpRight size={16} />
            </button>
          </motion.article>
          <motion.article
            className="studio-feature rose"
            initial={reduced ? false : { opacity: 0, y: 26 }}
            whileInView={reduced ? undefined : { opacity: 1, y: 0 }}
            viewport={{ once: true, amount: 0.15 }}
            transition={{ duration: 0.55, ease: "easeOut" }}
          >
            <div className="feature-mini editorial-feature-art">
              <img src="/images/editorial/feature-make.webp" alt="" loading="lazy" decoding="async" />
              <span className="editorial-feature-tag">
                <FileText size={14} /> Make something new
              </span>
            </div>
            <span className="feature-step">03 / MAKE</span>
            <h3>Make it. Then make it yours.</h3>
            <p>
              Create a code, a color or a fresh idea. Copy your result or
              download it for later.
            </p>
            <button onClick={() => onOpen("qr")}>
              Make a QR code <ArrowUpRight size={16} />
            </button>
          </motion.article>
        </div>
      </section>
      <section
        className="studio-human-connection"
        aria-labelledby="human-connection-title"
      >
        <div className="container human-connection-layout">
          <div className="human-connection-copy">
            <span className="studio-kicker">HUMAN IDEAS, THOUGHTFUL TECHNOLOGY</span>
            <h2 id="human-connection-title">
              Made for your ideas.
              <br />
              Not the other way around.
            </h2>
            <p>
              The best tools get out of your way, so you can focus on what
              matters: making something good.
            </p>
          </div>
          <div className="human-connection-visual" aria-hidden="true">
            <img
              src="/images/editorial/human-ai-hands.webp"
              alt=""
              loading="lazy"
              decoding="async"
            />
          </div>
        </div>
      </section>
      <section className="studio-tour container" aria-labelledby="tour-title">
        <span className="studio-kicker">A SMALL WORKSPACE. A BIG EXHALE.</span>
        <h2 id="tour-title">
          You don’t need a complicated workflow.
          <br />
          Just a little room to work.
        </h2>
        <p>Useful controls. Clear results. A shortcut that feels like yours.</p>
        <motion.div
          className="tour-frame"
          initial={reduced ? false : { opacity: 0, y: 34, scale: 0.985 }}
          whileInView={reduced ? undefined : { opacity: 1, y: 0, scale: 1 }}
          viewport={{ once: true, amount: 0.2 }}
          transition={{ duration: 0.7, ease: "easeOut" }}
        >
          <div className="tour-browser">
            <div className="tour-browser-bar">
              <span>
                <i />
                <i />
                <i />
              </span>
              <span>your little workspace</span>
              <LockKeyhole size={13} />
            </div>
            <div className="tour-app">
              <div className="tour-heading">
                <span>
                  <Braces size={19} /> JSON formatter
                </span>
                <span className="tour-local">
                  Runs locally <span />
                </span>
              </div>
              <div className="tour-panels">
                <div>
                  <span className="tour-panel-label">A little messy</span>
                  <pre>
                    {'{name:"WeMe", tools:' + count + ",\n ready:true,}"}
                  </pre>
                  <span className="tour-hint">
                    Bring what you’re working on.
                  </span>
                </div>
                <div>
                  <span className="tour-panel-label">
                    A little clearer <Check size={14} />
                  </span>
                  <pre>
                    {'{\n  "name": "WeMe",\n  "tools": ' +
                      count +
                      ',\n  "ready": true\n}'}
                  </pre>
                  <span className="tour-hint">
                    <Copy size={13} /> Copy <Download size={13} /> Download
                  </span>
                </div>
              </div>
            </div>
          </div>
          <img
            className="tour-editorial-detail"
            src="/images/editorial/workspace-details.webp"
            alt=""
            loading="lazy"
            decoding="async"
          />
          <div className="tour-sticker">
            <Sparkles size={20} /> Less busywork.
            <br />
            More good stuff.
          </div>
        </motion.div>
        <button className="button studio-button" onClick={() => onOpen("json")}>
          Try the workspace <ArrowRight size={16} />
        </button>
      </section>
      <section className="studio-picks container" aria-labelledby="picks-title">
        <div className="studio-section-heading">
          <div>
            <span className="studio-kicker">
              MADE FOR LITTLE MOMENTS OF PROGRESS
            </span>
            <h2 id="picks-title">
              You bring the idea.
              <br />
              We’ll bring the little tools.
            </h2>
          </div>
          <p>
            A few good places to start.
            <br />
            The rest is up to you.
          </p>
        </div>
        <div className="studio-pick-grid">
          {[
            {
              kind: "check" as const,
              id: "email",
              title: "A little confidence",
              text: "Check an address before you hit send.",
              action: "Verify an email",
            },
            {
              kind: "tidy" as const,
              id: "markdown",
              title: "A little clarity",
              text: "Give your next README a proper preview.",
              action: "Preview Markdown",
            },
            {
              kind: "make" as const,
              id: "qr",
              title: "A little possibility",
              text: "Turn a link into something worth sharing.",
              action: "Create a QR code",
            },
          ].map((item) => (
            <motion.button
              className="studio-pick"
              key={item.id}
              initial={reduced ? false : { opacity: 0, y: 24 }}
              whileInView={reduced ? undefined : { opacity: 1, y: 0 }}
              viewport={{ once: true, amount: 0.2 }}
              transition={{ duration: 0.5, ease: "easeOut" }}
              onClick={() => onOpen(item.id)}
            >
              <span className="pick-title">
                {item.title}
                <ArrowUpRight size={17} />
              </span>
              <ToolkitArt kind={item.kind} />
              <span className="pick-description">{item.text}</span>
              <span className="pick-action">
                {item.action} <ArrowRight size={14} />
              </span>
            </motion.button>
          ))}
        </div>
      </section>
    </div>
  );
}

export function LandingOutro({
  count,
  localCount,
  onSection,
}: {
  count: number;
  localCount: number;
  onSection: (id: string) => void;
}) {
  return (
    <div className="studio-landing">
      <section
        className="studio-pass"
        id="open-source"
        aria-labelledby="pass-title"
      >
        <div className="studio-pass-inner container">
          <img
            className="pass-editorial-art"
            src="/images/editorial/open-source-glass.webp"
            alt=""
            loading="lazy"
            decoding="async"
          />
          <span className="studio-kicker">THE EVERYDAY PASS</span>
          <h2 id="pass-title">Good tools are for everyone.</h2>
          <p>
            No subscriptions. No sign-up. Just a little help when you need it.
          </p>
          <div className="pass-card">
            <div className="pass-card-paper">
              <span className="pass-spark">
                <Sparkles size={19} /> WeMe, all yours
              </span>
              <strong>
                Free<span>to make things happen</span>
              </strong>
              <ul>
                <li>
                  <Check size={13} /> All {count} tools in one collection
                </li>
                <li>
                  <Check size={13} /> Examples to get you started
                </li>
                <li>
                  <Check size={13} /> Copy and download your results
                </li>
                <li>
                  <Check size={13} /> Open-source foundations
                </li>
              </ul>
            </div>
            <button
              className="button studio-button"
              onClick={() => onSection("tools")}
            >
              Find your next shortcut <ArrowRight size={15} />
            </button>
          </div>
          <a
            className="studio-source-link"
            href="https://github.com/8dazo/valid-mail"
            target="_blank"
            rel="noopener noreferrer"
          >
            Built in the open. Explore the source <GitFork size={15} />
          </a>
        </div>
      </section>
      <section
        className="studio-faq container"
        id="about"
        aria-labelledby="faq-title"
      >
        <div>
          <span className="studio-kicker">A FEW LITTLE ANSWERS</span>
          <h2 id="faq-title">
            Good questions.
            <br />
            Simple answers.
          </h2>
          <p>
            A calmer corner for builders, writers and curious people. Here’s how
            it works.
          </p>
        </div>
        <div className="faq-list">
          {[
            {
              q: "Can I use WeMe for free?",
              a: "Yes. Every tool is free to use, with no account or subscription needed.",
            },
            {
              q: "Does my data leave my browser?",
              a: `The ${localCount} local tools run in your browser. Email verification and company-email discovery send the details needed for their checks to the Go server. Your inputs aren’t saved in browser storage.`,
            },
            {
              q: "How do I get started?",
              a: "Choose a tool, paste your input or import a supported file, then run it. Every tool has an example, and you can copy or download your result.",
            },
            {
              q: "Does email verification prove a mailbox exists?",
              a: "Address and domain checks are separate from mailbox checks. SMTP needs a reachable mail-server route; a blocked or incomplete check leaves mailbox existence unknown.",
            },
            {
              q: "Can I host WeMe myself?",
              a: "Yes. The GitHub README includes local, Docker, Render and Vercel setup instructions, along with the Go libraries used by each tool.",
            },
          ].map((item, index) => (
            <details key={item.q} open={index === 0}>
              <summary>
                {item.q}
                <ChevronDown size={16} />
              </summary>
              <p>{item.a}</p>
            </details>
          ))}
        </div>
      </section>
    </div>
  );
}

export function StudioFooter({
  brand,
  onSection,
  onOpen,
}: {
  brand: ReactNode;
  onSection: (id: string) => void;
  onOpen: OpenTool;
}) {
  return (
    <footer className="studio-footer">
      <div className="container">
        <div className="studio-footer-top">
          <div className="footer-brand">
            {brand}
            <p>
              Little tools.
              <br />A little more possibility.
            </p>
            <a
              href="https://github.com/8dazo/valid-mail"
              aria-label="WeMe on GitHub"
              target="_blank"
              rel="noopener noreferrer"
            >
              <GitFork size={18} /> Built in the open <ArrowUpRight size={14} />
            </a>
          </div>
          <div>
            <h3>The toolkit</h3>
            <button onClick={() => onSection("tools")}>
              Explore all tools
            </button>
            <button onClick={() => onOpen("email")}>Email verifier</button>
            <button onClick={() => onOpen("json")}>JSON formatter</button>
            <button onClick={() => onOpen("markdown")}>README preview</button>
          </div>
          <div>
            <h3>Around here</h3>
            <button onClick={() => onSection("how-it-works")}>
              How it works
            </button>
            <button onClick={() => onSection("about")}>About WeMe</button>
            <button onClick={() => onSection("about")}>Common questions</button>
          </div>
          <div>
            <h3>Good foundations</h3>
            <a
              href="https://github.com/8dazo/valid-mail"
              target="_blank"
              rel="noopener noreferrer"
            >
              GitHub <ArrowUpRight size={12} />
            </a>
            <a
              href="https://github.com/8dazo/valid-mail/blob/main/README.md"
              target="_blank"
              rel="noopener noreferrer"
            >
              Documentation
            </a>
            <a
              href="https://github.com/8dazo/valid-mail/blob/main/docs/third-party.md"
              target="_blank"
              rel="noopener noreferrer"
            >
              Library credits
            </a>
          </div>
        </div>
      </div>
      <div className="footer-panorama">
        <img
          className="footer-panorama-image"
          src="/images/editorial/footer-panorama.webp"
          alt=""
          loading="lazy"
          decoding="async"
        />
        <div className="container footer-panorama-content">
          <div className="footer-wordmark" aria-hidden="true">
          <span>W</span>
          <span>e</span>
          <span>M</span>
          <span>e</span>
          <i>
            <Sparkles strokeWidth={1.2} />
          </i>
        </div>
          <div className="studio-footer-bottom">
            <span>A little less friction. A little more you.</span>
            <span>© {new Date().getFullYear()} WeMe</span>
          </div>
        </div>
      </div>
    </footer>
  );
}

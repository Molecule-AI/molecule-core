import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

export function middleware(request: NextRequest) {
  // Redirect /en, /zh, etc. locale prefixes back to root
  const pathname = request.nextUrl.pathname;
  const locales = /^\/(en|zh|ja|ko|fr|de|es|pt|it|ru|ar|hi|th|vi|nl|sv|da|nb|fi|pl|cs|tr|uk|he|id|ms)(\/|$)/;
  if (locales.test(pathname)) {
    return NextResponse.redirect(new URL("/", request.url));
  }
}

export const config = {
  matcher: ["/((?!api|_next/static|_next/image|favicon.ico).*)"],
};

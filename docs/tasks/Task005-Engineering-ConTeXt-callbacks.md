<header><title>Task 005 - Engineering ConTeXt</title></header>

# Task 005 - Showing that ConTeXt callbacks can be used to reduce computation

1. **completed** Find Lua callbacks to stop pdf generation

2. **completed** Find Lua callbacks to stop page setting

## Architecture

We plan to use the LuaMetaTex callbacks, found in chapter 10 of the 
[LuaMetaTeX Reference 
Manual](http://www.pragma-ade.com/general/manuals/luametatex.pdf),
to stop the effective computation after the following ConTeXt tasks:

1. After all macro expansions (and hence after *my* calls into Lua)
   but before line/paragraph/page layout begins.

2. After line/paragraph/page layout but before PDF generation.

3. After all PDF generated (ie. a "normal" "full" ConTeXt run).

Stopping after all macro expansions would allow my code generation
builds to proceed without the un-needed page setting or PDF generation.

Stopping after the line/paragraph/page layout would allow multiple
"faster(?)" ConTeXt runs while the "*.tuc" file converges to a complete
set of page numbers and cross references (etc). Then, once the "*.tuc"
file has converged, a full ConTeXt run with PDF output could be
done.

*Internally* ConTeXt is structured as a tight pipeline with each of the 
"traditional" TeX stages "Mouth", "Stomach", "page setting", PDF 
generation.... tightly "chained"... This means that there is no "one" 
place in the code where all macro expansions have completed but before the 
page setting "starts", or similarly, after the page setting has finished 
but before the PDF generation "starts". 

See Hans Hagen's reply: [[NTG-context] Using ConTeXt-LMTX for modern 
Mathematically-Literate-Programming 1/2 Hans 
Hagen](https://mailman.ntg.nl/pipermail/ntg-context/2020/100481.html).

Some example code that I placed just before `\startdocument` or 
`\startcomponent` 

```
\def\test{
  \directlua{
    print("Hello from test")
  }
}

\startlua
print("Hello from lua")

function append_to_vlist_filter()
  return nil
end

callbacks.register("append_to_vlist_filter", append_to_vlist_filter)
\stoplua
```

## Problems

I had problems getting the `\startlua` `\stoplua` working. The primary 
problem was un-observed Lua compilation failures. 

## Reflections

**luametatex document** 

Using `pre_output_filter` to return false, empties the page (except for 
headers/footers) and saves about 6 seconds (out of 18 seconds). **BUT** 
the `luametatex.tuc` is ruined. As Hans Hagen suggested: 

> but keep in mind that multipass data is flushed as part of the shipout 
> (because it is often location and order bound) 

Using `append_to_vlist_filter` to return nil, empties (nearly) everything 
(but seems to yield a blank page per chapter). This saves 8 seconds (out 
of 18). **BUT** as expcected the `luametatex.tuc` is ruined. 

However, TeX macros are still run (so a simple \test macro called into Lua 
and printed "Hello from test").

**luametatex document** 

Using `append_to_vlist_filter` to return nil, empties (nearly) everything 
(but seems to yield a blank page per chapter). This saves 6 seconds (out 
of 190). **BUT** as expcected the `luametatex.tuc` is ruined. 

**Overall**

These (rather crude) attempts at "stopping" computation at various points 
in the TeX pipeline do not actually show much benefit.

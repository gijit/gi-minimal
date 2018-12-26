--  depend.lua:
--
--  Implement Depth-First-Search (DFS)
--  on the graph of depedencies
--  between types. A pre-order
--  traversal will print
--  leaf types before the compound
--  types that need them defined.

local __dfsTestMode = false


function __isBasicTyp(typ)
   if typ == nil or
   typ.kind == nil then
      return false
   end
   
   -- we can skip all basic types,
   -- as they are already defined.
   --
   if typ.kind <= 16 or -- __kindComplex128 or earlier primitive.
      typ.kind == 24 or -- __kindString
      typ.kind == 26 or -- __kindUnsafePointer
   (typ.kind == 20 and typ.__str == "interface {}") then -- empty interface

      return true
   end
   return false
end

-- a function on nodes to force instantiation of
-- any types this node depends on, i.e. those
-- types (not values) that were described but
-- lazily instantated. Calls self.typ.bloom
-- on our subtree in depth-first order.
--
local function __makeRequiredTypes(self)
   if self.made then return end;
   self.made = true;
   if __isBasicTyp(self.typ) then
      return -- basic types are always leaf nodes, no children.
   end
   
   if self.children then
      for _, ch in ipairs(self.children) do
         __makeRequiredTypes(ch)
      end
   end
   self.typ.bloom(self.typ)
end

function __newDfsNode(self, name, typ)
   if typ == nil then
      error "typ cannot be nil in __newDfsNode"
   end
   if not __dfsTestMode then
      if typ.__str == nil then
         print(debug.traceback())
         error "typ must be typ, in __newDfsNode"
      end
      -- but we won't know the kind until
      -- later, since this may be early in
      -- typ construction.
   end
   
   local nd = self.dfsDedup[typ]
   if nd ~= nil then
      return nd
   end
   local node= {
      visited=false,
      children=false, -- lazily convert 'children' to a table upon first addition, for better printing.
      dedupChildren={},
      id = self.dfsNextID,
      name=name,
      typ=typ,
      made=false, -- set to true when a value using the type is instantiated. i.e. when tfun is called.
      makeRequiredTypes = __makeRequiredTypes,
   }
   self.dfsNextID=self.dfsNextID+1
   self.dfsDedup[typ] = node
   table.insert(self.dfsNodes, node)

   --print("just added to dfsNodes node "..name)
   --__st(typ, "typ, in __newDfsNode")
   
   self.stale = true
   
   return node
end



-- par should be a node; e.g. typ.__dfsNode
function __addChild(self, parTyp, chTyp)

   if parTyp == nil then
      error "parTyp cannot be nil in __addChild"
   end
   if chTyp == nil then
      print(debug.traceback())
      error "chTyp cannot be nil in __addChild"
   end
   if not __dfsTestMode then
      if parTyp.__str == nil then
         print(debug.traceback())
         error "parTyp must be typ, in __addChild"
      end
      if chTyp.__str == nil then
         print(debug.traceback())
         error "chTyp must be typ, in __addChild"
      end
   end

   -- we can skip all basic types,
   -- as they are already defined.   
   if __isBasicTyp(chTyp) then
      return
   end
   if __isBasicTyp(parTyp) then
      error("__addChild error: parent was basic type. "..
               "cannot add child to basic typ ".. parType.__str)
   end

   local chNode = self.dfsDedup[chTyp]
   if chNode == nil then
      -- child was previously generated, so
      -- we don't need to worry about this
      -- dependency
      return
   end
   
   local parNode = self.dfsDedup[parTyp]
   if parNode == nil then
      parNode = self:newDfsNode(parTyp.__str, parTyp)
   end
   
   if parNode.dedupChildren[ch] ~= nil then
      -- avoid adding same child twice.
      return
   end

   -- In Lua both nil and the boolean
   -- value false represent false in
   -- a logical expression
   --
   if not parNode.children then
      -- we lazily instantiate children
      -- for better diagnostics. Its
      -- much clearer to see "children = false"
      -- than "children = 'table: 0x04e99af8'"
      parNode.children = {}
   end
   
   local pnc = #parNode.children
   parNode.dedupChildren[chNode]= pnc+1
   table.insert(parNode.children, chNode)
   self.stale = true
end

function __markGraphUnVisited(self)
   self.dfsOrder = {}
   for _,n in ipairs(self.dfsNodes) do
      n.visited = false
   end
   self.stale = false
end

function __emptyOutGraph(self)
   self.dfsOrder = {}
   self.dfsNodes = {} -- node stored in value.
   self.dfsDedup = {} -- payloadTyp key -> node value.
   self.dfsNextID = 0
   self.stale = false
end

function __dfsHelper(self, node)
   if node == nil then
      return
   end
   if node.visited then
      return
   end
   node.visited = true
   __st(node,"node, in __dfsHelper")
   if node.children then
      for _, ch in ipairs(node.children) do
         self:dfsHelper(ch)
      end
   end
   print("post-order visit sees node "..tostring(node.id).." : "..node.name)
   table.insert(self.dfsOrder, node)
end

function __showDFSOrder(self)
   if self.stale then
      self:doDFS()
   end
   for i, n in ipairs(self.dfsOrder) do
      print("dfs order "..i.." is "..tostring(n.id).." : "..n.name)
   end
end

function __doDFS(self)
   __markGraphUnVisited(self)
   for _, n in ipairs(self.dfsNodes) do
      self:dfsHelper(n)
   end
   self.stale = false
end

function __hasTypes(self)
   return self.dfsNextID ~= 0
end


function __NewDFSState()
   return {
      dfsNodes = {},
      dfsOrder = {},
      dfsDedup = {},
      dfsNextID = 0,

      doDFS = __doDFS,
      dfsHelper = __dfsHelper,
      reset = __emptyOutGraph,
      newDfsNode = __newDfsNode,
      addChild = __addChild,
      markGraphUnVisited = __markGraphUnVisited,
      hasTypes = __hasTypes,
      showDFSOrder=__showDFSOrder,
   }
end

--[[
-- test. To test, change the --[[ above to ---[[
--       and issue dofile('dfs.lua')
dofile 'tutil.lua' -- must be in prelude dir to test.

function __testDFS()
   __dfsTestMode = true
   local s = __NewDFSState()

   -- verify that reset()
   -- works by doing two passes.
   
   for i =1,2 do
      s:reset()
      
      local aPayload = {}
      local a = s:newDfsNode("a", aPayload)
   
      local adup = s:newDfsNode("a", aPayload)
      if adup ~= a then
          error "dedup failed."
      end

      local b = s:newDfsNode("b", {})
      local c = s:newDfsNode("c", {})
      local d = s:newDfsNode("d", {})
      local e = s:newDfsNode("e", {})
      local f = s:newDfsNode("f", {})

      -- separate island:
      local g = s:newDfsNode("g", {})
      
      s:addChild(a, b)

      -- check dedup of child add
      local startCount = #a.children
      s:addChild(a, b)
      if #a.children ~= startCount then
          error("child dedup failed.")
      end

      s:addChild(b, c)
      s:addChild(b, d)
      s:addChild(d, e)
      s:addChild(d, f)

      s:doDFS()

      s:showDFSOrder()

      __expectEq(s.dfsOrder[1], c)
      __expectEq(s.dfsOrder[2], e)
      __expectEq(s.dfsOrder[3], f)
      __expectEq(s.dfsOrder[4], d)
      __expectEq(s.dfsOrder[5], b)
      __expectEq(s.dfsOrder[6], a)
      __expectEq(s.dfsOrder[7], g)
   end
   
end
__testDFS()
__testDFS()
--]]
